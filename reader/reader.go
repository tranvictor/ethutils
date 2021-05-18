package reader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eu "github.com/tranvictor/ethutils"
)

var (
	DEFAULT_ADDRESS string = "0x0000000000000000000000000000000000000000"
)

const (
	DEFAULT_ETHERSCAN_APIKEY string = "UBB257TI824FC7HUSPT66KZUMGBPRN3IWV"
	DEFAULT_BSCSCAN_APIKEY   string = "62TU8Z81F7ESNJT38ZVRBSX7CNN4QZSP5I"
	DEFAULT_TOMOSCAN_APIKEY  string = ""
)

type EthReader struct {
	chain             string
	nodes             map[string]EthereumNode
	latestGasPrice    float64
	gasPriceTimestamp int64
	gpmu              sync.Mutex

	etherscanAPIKey string
	bscscanAPIKey   string
	tomoscanAPIKey  string
}

func newEthReaderGeneric(nodes map[string]string, chain string) *EthReader {
	ns := map[string]EthereumNode{}
	for name, c := range nodes {
		ns[name] = NewOneNodeReader(name, c)
	}
	return &EthReader{
		chain:             chain,
		nodes:             ns,
		latestGasPrice:    0.0,
		gasPriceTimestamp: 0,
		gpmu:              sync.Mutex{},
		etherscanAPIKey:   DEFAULT_ETHERSCAN_APIKEY,
		bscscanAPIKey:     DEFAULT_BSCSCAN_APIKEY,
		tomoscanAPIKey:    DEFAULT_TOMOSCAN_APIKEY,
	}
}

func NewBSCReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "bsc")
}

func NewBSCTestnetReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "bsc-test")
}

func NewKovanReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "kovan")
}

func NewRinkebyReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "rinkeby")
}

func NewRopstenReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "ropsten")
}

func NewKovanReader() *EthReader {
	nodes := map[string]string{
		"kovan-infura": "https://kovan.infura.io/v3/247128ae36b6444d944d4c3793c8e3f5",
	}
	return NewKovanReaderWithCustomNodes(nodes)
}

func NewRinkebyReader() *EthReader {
	nodes := map[string]string{
		"rinkeby-infura": "https://rinkeby.infura.io/v3/247128ae36b6444d944d4c3793c8e3f5",
	}
	return NewRinkebyReaderWithCustomNodes(nodes)
}

func NewBSCReader() *EthReader {
	nodes := map[string]string{
		"binance":  "https://bsc-dataseed.binance.org",
		"defibit":  "https://bsc-dataseed1.defibit.io",
		"ninicoin": "https://bsc-dataseed1.ninicoin.io",
	}
	return NewBSCReaderWithCustomNodes(nodes)
}

func NewBSCTestnetReader() *EthReader {
	nodes := map[string]string{
		"binance1": "https://data-seed-prebsc-1-s1.binance.org:8545",
		"binance2": "https://data-seed-prebsc-2-s1.binance.org:8545",
		"binance3": "https://data-seed-prebsc-1-s2.binance.org:8545",
	}
	return NewBSCReaderWithCustomNodes(nodes)
}

func NewRopstenReader() *EthReader {
	nodes := map[string]string{
		"ropsten-infura": "https://ropsten.infura.io/v3/247128ae36b6444d944d4c3793c8e3f5",
	}
	return NewRopstenReaderWithCustomNodes(nodes)
}

func NewTomoReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "tomo")
}

func NewTomoReader() *EthReader {
	nodes := map[string]string{
		"mainnet-tomo": "https://rpc.tomochain.com",
	}
	return NewTomoReaderWithCustomNodes(nodes)
}

func NewEthReaderWithCustomNodes(nodes map[string]string) *EthReader {
	return newEthReaderGeneric(nodes, "ethereum")
}

func NewEthReader() *EthReader {
	nodes := map[string]string{
		"mainnet-alchemy": "https://eth-mainnet.alchemyapi.io/jsonrpc/YP5f6eM2wC9c2nwJfB0DC1LObdSY7Qfv",
		"mainnet-infura":  "https://mainnet.infura.io/v3/247128ae36b6444d944d4c3793c8e3f5",
	}
	return NewEthReaderWithCustomNodes(nodes)
}

func errorInfo(errs []error) string {
	estrs := []string{}
	for i, e := range errs {
		estrs = append(estrs, fmt.Sprintf("%d. %s", i+1, e))
	}
	return strings.Join(estrs, "\n")
}

func wrapError(e error, name string) error {
	if e == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", name, e)
}

func (self *EthReader) SetEtherscanAPIKey(key string) {
	self.etherscanAPIKey = key
}

func (self *EthReader) SetBSCScanAPIKey(key string) {
	self.bscscanAPIKey = key
}

func (self *EthReader) SetTomoScanAPIKey(key string) {
	self.tomoscanAPIKey = key
}

type estimateGasResult struct {
	Gas   uint64
	Error error
}

func (self *EthReader) EstimateExactGas(from, to string, priceGwei float64, value *big.Int, data []byte) (uint64, error) {
	resCh := make(chan estimateGasResult, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			gas, err := n.EstimateGas(from, to, priceGwei, value, data)
			resCh <- estimateGasResult{
				Gas:   gas,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Gas, result.Error
		}
		errs = append(errs, result.Error)
	}
	return 0, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

func (self *EthReader) EstimateGas(from, to string, priceGwei, value float64, data []byte) (uint64, error) {
	return self.EstimateExactGas(from, to, priceGwei, eu.FloatToBigInt(value, 18), data)
}

type getCodeResponse struct {
	Code  []byte
	Error error
}

func (self *EthReader) GetCode(address string) (code []byte, err error) {
	resCh := make(chan getCodeResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			code, err := n.GetCode(address)
			resCh <- getCodeResponse{
				Code:  code,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Code, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

func (self *EthReader) TxInfoFromHash(tx string) (eu.TxInfo, error) {
	txObj, isPending, err := self.TransactionByHash(tx)
	if err != nil {
		return eu.TxInfo{"error", nil, nil, nil}, err
	}
	if txObj == nil {
		return eu.TxInfo{"notfound", nil, nil, nil}, nil
	} else {
		if isPending {
			return eu.TxInfo{"pending", txObj, nil, nil}, nil
		} else {
			receipt, _ := self.TransactionReceipt(tx)
			if receipt == nil {
				return eu.TxInfo{"pending", txObj, nil, nil}, nil
			} else {
				// only byzantium has status field at the moment
				// mainnet, ropsten are byzantium, other chains such as
				// devchain, kovan are not.
				// if PostState is a hash, it is pre-byzantium and all
				// txs with PostState are considered done
				if len(receipt.PostState) == len(common.Hash{}) {
					return eu.TxInfo{"done", txObj, []eu.InternalTx{}, receipt}, nil
				} else {
					if receipt.Status == 1 {
						// successful tx
						return eu.TxInfo{"done", txObj, []eu.InternalTx{}, receipt}, nil
					}
					// failed tx
					return eu.TxInfo{"reverted", txObj, []eu.InternalTx{}, receipt}, nil
				}
			}
		}
	}
}

type ksresponse struct {
	Data struct {
		Fast     string
		Standard string
		Low      string
		Default  string
	}
	Success bool
}

func (self *EthReader) RecommendedGasPriceFromKyberSwap() (low, average, fast float64, err error) {
	resp, err := http.Get("https://production-cache.kyber.network/gasPrice")
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}
	prices := ksresponse{}
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return 0, 0, 0, err
	}
	if !prices.Success {
		return 0, 0, 0, fmt.Errorf("failed response from kyberswap")
	}

	fastFloat, err := strconv.ParseFloat(prices.Data.Fast, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	standardFloat, err := strconv.ParseFloat(prices.Data.Standard, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	lowFloat, err := strconv.ParseFloat(prices.Data.Low, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return lowFloat, standardFloat, fastFloat, nil
}

// gas station response
type gsresponse struct {
	Average float64 `json:"average"`
	Fast    float64 `json:"fast"`
	Fastest float64 `json:"fastest"`
	SafeLow float64 `json:"safeLow"`
}

func (self *EthReader) RecommendedGasPriceFromEthGasStation(link string) (low, average, fast float64, err error) {
	resp, err := http.Get(link)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}
	prices := gsresponse{}
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return 0, 0, 0, err
	}
	return prices.SafeLow / 10, prices.Average / 10, prices.Fast / 10, nil
}

// {"status":"1","message":"OK","result":{"LastBlock":"11210958","SafeGasPrice":"79","ProposeGasPrice":"88","FastGasPrice":"104"}}
type etherscanGasResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		LastBlock       string `json:"LastBlock"`
		SafeGasPrice    string `json:"SafeGasPrice"`
		ProposeGasPrice string `json:"ProposeGasPrice"`
		FastGasPrice    string `json:"FastGasPrice"`
	} `json:"result"`
}

func (self *EthReader) RecommendedGasPriceFromEtherscan(link string) (low, average, fast float64, err error) {
	resp, err := http.Get(link)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}
	prices := etherscanGasResponse{}
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return 0, 0, 0, err
	}
	low, err = strconv.ParseFloat(prices.Result.SafeGasPrice, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	average, err = strconv.ParseFloat(prices.Result.ProposeGasPrice, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	fast, err = strconv.ParseFloat(prices.Result.FastGasPrice, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	return low, average, fast, nil
}

func (self *EthReader) RecommendedGasPriceKovan() (float64, error) {
	return 50, nil
}

func (self *EthReader) RecommendedGasPriceRinkeby() (float64, error) {
	return 50, nil
}

func (self *EthReader) RecommendedGasPriceRopsten() (float64, error) {
	return 50, nil
}

func (self *EthReader) RecommendedGasPriceTomo() (float64, error) {
	return 1, nil
}

func (self *EthReader) RecommendedGasPriceBSC() (float64, error) {
	return 10, nil
}

func (self *EthReader) RecommendedGasPriceBSCTestnet() (float64, error) {
	return 10, nil
}

func (self *EthReader) RecommendedGasPriceEthereum() (float64, error) {
	self.gpmu.Lock()
	defer self.gpmu.Unlock()
	if self.latestGasPrice == 0 || time.Now().Unix()-self.gasPriceTimestamp > 30 {
		// TODO
		// _, _, gsFast, err1 := self.RecommendedGasPriceFromEthGasStation("https://ethgasstation.info/json/ethgasAPI.json")
		// _, _, esFast, err2 := self.RecommendedGasPriceFromEtherscan("https://api.etherscan.io/api?module=gastracker&action=gasoracle&apikey=UBB257TI824FC7HUSPT66KZUMGBPRN3IWV")
		_, _, esFast, err3 := self.RecommendedGasPriceFromKyberSwap()
		if err3 != nil {
			return 0, fmt.Errorf("etherscan gas price lookup failed: %s", err3)
		}

		// if err1 != nil && err2 != nil {
		// 	return 0, fmt.Errorf("eth gas station gas price lookup failed: %s, etherscan gas price lookup failed: %s", err1, err2)
		// }

		self.latestGasPrice = esFast + 10
		self.gasPriceTimestamp = time.Now().Unix()
	}
	return self.latestGasPrice, nil
}

// return gwei
func (self *EthReader) RecommendedGasPrice() (float64, error) {
	switch self.chain {
	case "ethereum":
		return self.RecommendedGasPriceEthereum()
	case "ropsten":
		return self.RecommendedGasPriceRopsten()
	case "kovan":
		return self.RecommendedGasPriceKovan()
	case "rinkeby":
		return self.RecommendedGasPriceRinkeby()
	case "tomo":
		return self.RecommendedGasPriceTomo()
	case "bsc":
		return self.RecommendedGasPriceBSC()
	case "bsc-test":
		return self.RecommendedGasPriceBSCTestnet()
	}
	return 0, fmt.Errorf("'%s' chain is not supported", self.chain)
}

type getBalanceResponse struct {
	Balance *big.Int
	Error   error
}

func (self *EthReader) GetBalance(address string) (balance *big.Int, err error) {
	resCh := make(chan getBalanceResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			balance, err := n.GetBalance(address)
			resCh <- getBalanceResponse{
				Balance: balance,
				Error:   wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Balance, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

type getNonceResponse struct {
	Nonce uint64
	Error error
}

func (self *EthReader) GetMinedNonce(address string) (nonce uint64, err error) {
	resCh := make(chan getNonceResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			nonce, err := n.GetMinedNonce(address)
			resCh <- getNonceResponse{
				Nonce: nonce,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Nonce, result.Error
		}
		errs = append(errs, result.Error)
	}
	return 0, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

func (self *EthReader) GetPendingNonce(address string) (nonce uint64, err error) {
	resCh := make(chan getNonceResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			nonce, err := n.GetPendingNonce(address)
			resCh <- getNonceResponse{
				Nonce: nonce,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Nonce, result.Error
		}
		errs = append(errs, result.Error)
	}
	return 0, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

type transactionReceiptResponse struct {
	Receipt *types.Receipt
	Error   error
}

func (self *EthReader) TransactionReceipt(txHash string) (receipt *types.Receipt, err error) {
	resCh := make(chan transactionReceiptResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			receipt, err := n.TransactionReceipt(txHash)
			resCh <- transactionReceiptResponse{
				Receipt: receipt,
				Error:   wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Receipt, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

type transactionByHashResponse struct {
	Tx        *eu.Transaction
	IsPending bool
	Error     error
}

func (self *EthReader) TransactionByHash(txHash string) (tx *eu.Transaction, isPending bool, err error) {
	resCh := make(chan transactionByHashResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			tx, ispending, err := n.TransactionByHash(txHash)
			resCh <- transactionByHashResponse{
				Tx:        tx,
				IsPending: ispending,
				Error:     wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Tx, result.IsPending, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, false, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

// TODO: this method can't utilize all of the nodes because the result reference
// will be written in parallel and it is not thread safe
// func (self *EthReader) Call(result interface{}, method string, args ...interface{}) error {
// 	for _, node := range self.nodes {
// 		return node.Call(result, method, args...)
// 	}
// 	return fmt.Errorf("no nodes to call")
// }

type readContractToBytesResponse struct {
	Data  []byte
	Error error
}

func (self *EthReader) ReadContractToBytes(atBlock int64, from string, caddr string, abi *abi.ABI, method string, args ...interface{}) ([]byte, error) {
	resCh := make(chan readContractToBytesResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			data, err := n.ReadContractToBytes(atBlock, from, caddr, abi, method, args...)
			resCh <- readContractToBytesResponse{
				Data:  data,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Data, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

func (self *EthReader) ReadHistoryContractWithABI(atBlock int64, result interface{}, caddr string, abi *abi.ABI, method string, args ...interface{}) error {
	responseBytes, err := self.ReadContractToBytes(
		int64(atBlock), DEFAULT_ADDRESS, caddr, abi, method, args...)
	if err != nil {
		return err
	}
	return abi.UnpackIntoInterface(result, method, responseBytes)
}

func (self *EthReader) ReadContractWithABIAndFrom(result interface{}, from string, caddr string, abi *abi.ABI, method string, args ...interface{}) error {
	responseBytes, err := self.ReadContractToBytes(-1, from, caddr, abi, method, args...)
	if err != nil {
		return err
	}
	return abi.UnpackIntoInterface(result, method, responseBytes)
}

func (self *EthReader) ReadContractWithABI(result interface{}, caddr string, abi *abi.ABI, method string, args ...interface{}) error {
	responseBytes, err := self.ReadContractToBytes(-1, DEFAULT_ADDRESS, caddr, abi, method, args...)
	if err != nil {
		return err
	}
	return abi.UnpackIntoInterface(result, method, responseBytes)
}

func (self *EthReader) ReadHistoryContract(atBlock int64, result interface{}, caddr string, method string, args ...interface{}) error {
	abi, err := self.GetABI(caddr)
	if err != nil {
		return err
	}
	return self.ReadHistoryContractWithABI(atBlock, result, caddr, abi, method, args...)
}

func (self *EthReader) ReadContract(result interface{}, caddr string, method string, args ...interface{}) error {
	abi, err := self.GetABI(caddr)
	if err != nil {
		return err
	}
	return self.ReadContractWithABI(result, caddr, abi, method, args...)
}

func (self *EthReader) HistoryERC20Balance(atBlock int64, caddr string, user string) (*big.Int, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return nil, err
	}
	result := big.NewInt(0)
	err = self.ReadHistoryContractWithABI(atBlock, &result, caddr, abi, "balanceOf", eu.HexToAddress(user))
	return result, err
}

func (self *EthReader) ERC20Balance(caddr string, user string) (*big.Int, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return nil, err
	}
	result := big.NewInt(0)
	err = self.ReadContractWithABI(&result, caddr, abi, "balanceOf", eu.HexToAddress(user))
	return result, err
}

func (self *EthReader) HistoryERC20Decimal(atBlock int64, caddr string) (int64, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return 0, err
	}
	var result uint8
	err = self.ReadHistoryContractWithABI(atBlock, &result, caddr, abi, "decimals")
	return int64(result), err
}

func (self *EthReader) ERC20Decimal(caddr string) (int64, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return 0, err
	}
	var result uint8
	err = self.ReadContractWithABI(&result, caddr, abi, "decimals")
	return int64(result), err
}

type headerByNumberResponse struct {
	Header *types.Header
	Error  error
}

func (self *EthReader) HeaderByNumber(number int64) (*types.Header, error) {
	resCh := make(chan headerByNumberResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			header, err := n.HeaderByNumber(number)
			resCh <- headerByNumberResponse{
				Header: header,
				Error:  wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Header, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

func (self *EthReader) HistoryERC20Allowance(atBlock int64, caddr string, owner string, spender string) (*big.Int, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return nil, err
	}
	result := big.NewInt(0)
	err = self.ReadHistoryContractWithABI(
		atBlock,
		&result, caddr, abi,
		"allowance",
		eu.HexToAddress(owner),
		eu.HexToAddress(spender),
	)
	return result, err
}

func (self *EthReader) ERC20Allowance(caddr string, owner string, spender string) (*big.Int, error) {
	abi, err := eu.GetERC20ABI()
	if err != nil {
		return nil, err
	}
	result := big.NewInt(0)
	err = self.ReadContractWithABI(
		&result, caddr, abi,
		"allowance",
		eu.HexToAddress(owner),
		eu.HexToAddress(spender),
	)
	return result, err
}

func (self *EthReader) AddressFromContract(contract string, method string) (*common.Address, error) {
	result := common.Address{}
	err := self.ReadContract(&result, contract, method)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type getLogsResponse struct {
	Logs  []types.Log
	Error error
}

// if toBlock < 0, it will query to the latest block
func (self *EthReader) GetLogs(fromBlock, toBlock int, addresses []string, topic string) ([]types.Log, error) {
	resCh := make(chan getLogsResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			logs, err := n.GetLogs(fromBlock, toBlock, addresses, topic)
			resCh <- getLogsResponse{
				Logs:  logs,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Logs, result.Error
		}
		errs = append(errs, result.Error)
	}
	return nil, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}

type getBlockResponse struct {
	Block uint64
	Error error
}

func (self *EthReader) CurrentBlock() (uint64, error) {
	resCh := make(chan getBlockResponse, len(self.nodes))
	for i, _ := range self.nodes {
		n := self.nodes[i]
		go func() {
			block, err := n.CurrentBlock()
			resCh <- getBlockResponse{
				Block: block,
				Error: wrapError(err, n.NodeName()),
			}
		}()
	}
	errs := []error{}
	for i := 0; i < len(self.nodes); i++ {
		result := <-resCh
		if result.Error == nil {
			return result.Block, result.Error
		}
		errs = append(errs, result.Error)
	}
	return 0, fmt.Errorf("Couldn't read from any nodes: %s", errorInfo(errs))
}
