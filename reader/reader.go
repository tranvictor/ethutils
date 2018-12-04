package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	eu "github.com/tranvictor/ethutils"
)

const TIMEOUT time.Duration = 4 * time.Second

var SharedReader *EthReader
var once sync.Once

type EthReader struct {
	clients map[string]*rpc.Client

	latestGasPrice    float64
	gasPriceTimestamp int64
	gpmu              sync.Mutex
}

func NewRopstenReader() *EthReader {
	nodes := map[string]string{
		"ropsten-infura": "https://ropsten.infura.io",
	}
	clients := map[string]*rpc.Client{}
	for name, c := range nodes {
		client, err := rpc.Dial(c)
		if err != nil {
			log.Printf("Couldn't connect to: %s - %v", c, err)
		} else {
			clients[name] = client
		}
	}
	return &EthReader{
		clients:           clients,
		latestGasPrice:    0.0,
		gasPriceTimestamp: 0,
		gpmu:              sync.Mutex{},
	}
}

func NewEthReader() *EthReader {
	once.Do(func() {
		nodes := map[string]string{
			"mainnet-alchemy":  "https://eth-mainnet.alchemyapi.io/jsonrpc/3QSu5K3-xUgD_1WThGHmxfhe8QmmdmCC",
			"mainnet-quiknode": "https://optionally-pleasant-horse.quiknode.io/9d72a0f8-0d8b-4e4c-aef1-eb529e05cdb9/V1ZsC_tuomfETYotFo4KKA==/",
			"mainnet-infura":   "https://mainnet.infura.io",
			"mainnet-kyber":    "https://semi-node.kyber.network",
		}
		clients := map[string]*rpc.Client{}
		for name, c := range nodes {
			client, err := rpc.Dial(c)
			if err != nil {
				log.Printf("Couldn't connect to: %s - %v", c, err)
			} else {
				clients[name] = client
			}
		}
		SharedReader = &EthReader{
			clients:           clients,
			latestGasPrice:    0.0,
			gasPriceTimestamp: 0,
			gpmu:              sync.Mutex{},
		}
	})
	return SharedReader
}

// gas station response
type abiresponse struct {
	Status  string `json:"string"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func (self *EthReader) GetABIFromFile(filename string) (*abi.ABI, error) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("couldn't get filepath of the caller")
	}
	content, err := ioutil.ReadFile(path.Join(path.Dir(current), filename))
	if err != nil {
		return nil, err
	}

	result, err := abi.JSON(strings.NewReader(string(content)))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (self *EthReader) GetABI(address string) (*abi.ABI, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getabi&address=%s&apikey=UBB257TI824FC7HUSPT66KZUMGBPRN3IWV", address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	abiresp := abiresponse{}
	err = json.Unmarshal(body, &abiresp)
	if err != nil {
		return nil, err
	}
	result, err := abi.JSON(strings.NewReader(abiresp.Result))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (self *EthReader) EstimateGas(from, to string, priceGwei, value float64, data []byte) (uint64, error) {
	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)
	price := eu.FloatToBigInt(priceGwei, 9)
	v := eu.FloatToBigInt(value, 18)
	errors := map[string]error{}
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		gas, err := ethcli.EstimateGas(timeout, ethereum.CallMsg{
			From:     fromAddr,
			To:       &toAddr,
			Gas:      0,
			GasPrice: price,
			Value:    v,
			Data:     data,
		})
		defer cancel()
		if err == nil {
			return gas, err
		} else {
			errors[name] = err
		}
	}
	return 0, makeError(errors)
}

func (self *EthReader) GetCode(address string) (code []byte, err error) {
	errors := map[string]error{}
	addr := common.HexToAddress(address)
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		code, err = ethcli.CodeAt(timeout, addr, nil)
		defer cancel()
		if err == nil {
			return code, nil
		} else {
			errors[name] = err
		}
	}
	return code, makeError(errors)
}

func (self *EthReader) TxInfoFromHash(tx string) (eu.TxInfo, error) {
	txObj, isPending, err := self.TransactionByHash(tx)
	if err != nil {
		return eu.TxInfo{"error", nil, nil}, err
	}
	if txObj == nil {
		return eu.TxInfo{"notfound", nil, nil}, nil
	} else {
		if isPending {
			return eu.TxInfo{"pending", txObj, nil}, nil
		} else {
			receipt, _ := self.TransactionReceipt(tx)
			if receipt == nil {
				return eu.TxInfo{"pending", txObj, nil}, nil
			} else {
				// only byzantium has status field at the moment
				// mainnet, ropsten are byzantium, other chains such as
				// devchain, kovan are not.
				// if PostState is a hash, it is pre-byzantium and all
				// txs with PostState are considered done
				if len(receipt.PostState) == len(common.Hash{}) {
					return eu.TxInfo{"done", txObj, receipt}, nil
				} else {
					if receipt.Status == 1 {
						// successful tx
						return eu.TxInfo{"done", txObj, receipt}, nil
					}
					// failed tx
					return eu.TxInfo{"reverted", txObj, receipt}, nil
				}
			}
		}
	}
}

// gas station response
type gsresponse struct {
	Average float64 `json:"average"`
	Fast    float64 `json:"fast"`
	Fastest float64 `json:"fastest"`
	SafeLow float64 `json:"safeLow"`
}

// return gwei
func (self *EthReader) RecommendedGasPrice() (float64, error) {
	if self.latestGasPrice == 0 || time.Now().Unix()-self.gasPriceTimestamp > 30 {
		resp, err := http.Get("https://ethgasstation.info/json/ethgasAPI.json")
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		prices := gsresponse{}
		err = json.Unmarshal(body, &prices)
		if err != nil {
			return 0, err
		}
		self.gpmu.Lock()
		defer self.gpmu.Unlock()
		self.latestGasPrice = float64(prices.Fast) / 10.0
		self.gasPriceTimestamp = time.Now().Unix()
	}
	return self.latestGasPrice, nil
}

func (self *EthReader) GetBalance(address string) (balance *big.Int, err error) {
  errors := map[string]error{}
	acc := common.HexToAddress(address)
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		balance, err = ethcli.BalanceAt(timeout, acc, nil)
		defer cancel()
		if err == nil {
			return balance, err
		} else {
			errors[name] = err
		}
	}
	return balance, makeError(errors)
}

func (self *EthReader) GetMinedNonce(address string) (nonce uint64, err error) {
  errors := map[string]error{}
	acc := common.HexToAddress(address)
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		nonce, err = ethcli.NonceAt(timeout, acc, nil)
		defer cancel()
		if err == nil {
			return nonce, err
		} else {
			errors[name] = err
		}
	}
	return nonce, makeError(errors)
}

func (self *EthReader) GetPendingNonce(address string) (nonce uint64, err error) {
  errors := map[string]error{}
	acc := common.HexToAddress(address)
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		nonce, err = ethcli.PendingNonceAt(timeout, acc)
		defer cancel()
		if err == nil {
			return nonce, err
		} else {
			errors[name] = err
		}
	}
	return nonce, makeError(errors)
}

func (self *EthReader) TransactionReceipt(txHash string) (receipt *types.Receipt, err error) {
	errors := map[string]error{}
	hash := common.HexToHash(txHash)
	for name, client := range self.clients {
		ethcli := ethclient.NewClient(client)
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		receipt, err = ethcli.TransactionReceipt(timeout, hash)
		defer cancel()
		if err == nil {
			return receipt, nil
		} else {
			errors[name] = err
		}
	}
	return receipt, makeError(errors)
}

func (self *EthReader) transactionByHashOnNode(ctx context.Context, hash common.Hash, client *rpc.Client) (tx *eu.Transaction, isPending bool, err error) {
	var json *eu.Transaction
	err = client.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, ethereum.NotFound
	} else if _, r, _ := json.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	return json, json.Extra.BlockNumber == nil, nil
}

func (self *EthReader) TransactionByHash(txHash string) (tx *eu.Transaction, isPending bool, err error) {
	errors := map[string]error{}
	hash := common.HexToHash(txHash)
	for name, client := range self.clients {
		// fmt.Printf("Start time: %s\n", time.Now())
		timeout, cancel := context.WithTimeout(context.Background(), TIMEOUT)
		tx, isPending, err = self.transactionByHashOnNode(timeout, hash, client)
		// fmt.Printf("End time: %s\n", time.Now())
		defer cancel()
		if err == nil {
			return tx, isPending, err
		} else {
			errors[name] = err
		}
	}
	return tx, isPending, makeError(errors)
}

func (self *EthReader) Call(result interface{}, method string, args ...interface{}) error {
	errors := map[string]error{}
	for name, client := range self.clients {
		timeout, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		err := client.CallContext(timeout, result, method, args)
		defer cancel()
		if err == nil {
			return nil
		} else {
			errors[name] = err
		}
	}
	return makeError(errors)
}

func (self *EthReader) readContractToBytes(caddr string, abi *abi.ABI, method string, args ...interface{}) ([]byte, error) {
	errors := map[string]error{}
	contract := eu.HexToAddress(caddr)
	data, err := abi.Pack(method, args...)
	if err != nil {
		return []byte{}, err
	}
	for name, client := range self.clients {
		timeout, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		ethcli := ethclient.NewClient(client)
		result, err := ethcli.CallContract(timeout, ethereum.CallMsg{
			From:     common.Address{},
			To:       &contract,
			Gas:      0,
			GasPrice: nil,
			Value:    nil,
			Data:     data,
		}, nil)
		defer cancel()
		if err == nil {
			return result, nil
		} else {
			errors[name] = err
		}
	}
	return []byte{}, makeError(errors)
}

func (self *EthReader) ReadContractWithABI(result interface{}, caddr string, abi *abi.ABI, method string, args ...interface{}) error {
	responseBytes, err := self.readContractToBytes(caddr, abi, method, args...)
	if err != nil {
		return err
	}
	return abi.Unpack(result, method, responseBytes)
}

func (self *EthReader) ReadContract(result interface{}, caddr string, method string, args ...interface{}) error {
	abi, err := self.GetABI(caddr)
	if err != nil {
		return err
	}
	return self.ReadContractWithABI(result, caddr, abi, method, args...)
}

func (self *EthReader) ERC20Balance(caddr string, user string) (*big.Int, error) {
	abi, err := self.GetABIFromFile("erc20.abi")
	if err != nil {
		return nil, err
	}
	result := big.NewInt(0)
	err = self.ReadContractWithABI(&result, caddr, abi, "balanceOf", eu.HexToAddress(user))
	return result, err
}

func (self *EthReader) ERC20Decimal(caddr string) (int64, error) {
	abi, err := self.GetABIFromFile("erc20.abi")
	if err != nil {
		return 0, err
	}
	var result uint8
	err = self.ReadContractWithABI(&result, caddr, abi, "decimals")
	return int64(result), err
}
