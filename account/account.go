package account

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tranvictor/ethutils"
	"github.com/tranvictor/ethutils/broadcaster"
	"github.com/tranvictor/ethutils/reader"
)

type Account struct {
	signer      Signer
	reader      *reader.EthReader
	broadcaster *broadcaster.Broadcaster
	address     common.Address
}

func (self *Account) Address() string {
	return self.address.Hex()
}

func (self *Account) GetMinedNonce() (uint64, error) {
	return self.reader.GetMinedNonce(self.Address())
}

func (self *Account) GetPendingNonce() (uint64, error) {
	return self.reader.GetPendingNonce(self.Address())
}

func (self *Account) ListOfPendingNonces() ([]uint64, error) {
	minedNonce, err := self.GetMinedNonce()
	if err != nil {
		return []uint64{}, err
	}
	pendingNonce, err := self.GetPendingNonce()
	if err != nil {
		return []uint64{}, err
	}
	result := []uint64{}
	for i := minedNonce; i < pendingNonce; i++ {
		result = append(result, i)
	}
	return result, nil
}

func (self *Account) SendETHWithNonceAndPrice(nonce uint64, priceGwei float64, ethAmount *big.Int, to string) (tx *types.Transaction, broadcasted bool, errors error) {
	tx = ethutils.BuildExactSendETHTx(nonce, to, ethAmount, priceGwei)
	signedTx, err := self.signer.SignTx(tx)
	if err != nil {
		return tx, false, fmt.Errorf("couldn't sign the tx: %s", err)
	}
	_, broadcasted, errors = self.broadcaster.BroadcastTx(signedTx)
	return signedTx, broadcasted, errors
}

func (self *Account) SendAllETHWithPrice(priceGwei float64, to string) (tx *types.Transaction, broadcasted bool, errors error) {
	nonce, err := self.GetMinedNonce()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get nonce: %s", err)
	}
	balance, err := self.reader.GetBalance(self.Address())
	if err != nil {
		return nil, false, fmt.Errorf("cannot get balance: %s", err)
	}
	amount := balance.Sub(balance, big.NewInt(0).Mul(big.NewInt(21000), ethutils.FloatToBigInt(priceGwei, 9)))
	if amount.Cmp(big.NewInt(0)) != 1 {
		return nil, false, fmt.Errorf("not enough to do a tx with gas price: %f gwei", priceGwei)
	}
	return self.SendETHWithNonceAndPrice(nonce, priceGwei, amount, to)
}

func (self *Account) SendAllETH(to string) (tx *types.Transaction, broadcasted bool, errors error) {
	nonce, err := self.GetMinedNonce()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get nonce: %s", err)
	}
	priceGwei, err := self.reader.RecommendedGasPrice()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get recommended gas price: %s", err)
	}
	balance, err := self.reader.GetBalance(self.Address())
	if err != nil {
		return nil, false, fmt.Errorf("cannot get balance: %s", err)
	}
	amount := balance.Sub(balance, big.NewInt(0).Mul(big.NewInt(21000), ethutils.FloatToBigInt(priceGwei, 9)))
	if amount.Cmp(big.NewInt(0)) != 1 {
		return nil, false, fmt.Errorf("not enough to do a tx with gas price: %f gwei", priceGwei)
	}
	return self.SendETHWithNonceAndPrice(nonce, priceGwei, amount, to)
}

func (self *Account) SendERC20(tokenAddr string, tokenAmount float64, to string) (tx *types.Transaction, broadcasted bool, errors error) {
	decimals, err := self.reader.ERC20Decimal(tokenAddr)
	if err != nil {
		return nil, false, fmt.Errorf("cannot get token decimal: %s", err)
	}
	amount := ethutils.FloatToBigInt(tokenAmount, decimals)
	return self.CallContract(0, tokenAddr, "transfer", ethutils.HexToAddress(to), amount)
}

func (self *Account) SendETH(ethAmount float64, to string) (tx *types.Transaction, broadcasted bool, errors error) {
	nonce, err := self.GetMinedNonce()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get nonce: %s", err)
	}
	priceGwei, err := self.reader.RecommendedGasPrice()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get recommended gas price: %s", err)
	}
	amount := ethutils.FloatToBigInt(ethAmount, 18)
	return self.SendETHWithNonceAndPrice(nonce, priceGwei, amount, to)
}

func (self *Account) SendETHToMultipleAddressesWithPrice(priceGwei float64, amounts []float64, addresses []string) (txs []*types.Transaction, broadcasteds []bool, errors []error) {
	if len(amounts) != len(addresses) {
		panic("amounts and addresses must have the same length")
		return
	}
	nonce, err := self.GetMinedNonce()
	if err != nil {
		panic(fmt.Errorf("cannot get nonce: %s", err))
		return
	}
	txs = []*types.Transaction{}
	broadcasteds = []bool{}
	errors = []error{}
	for i, addr := range addresses {
		amount := amounts[i]
		newNonce := nonce + uint64(i)
		tx, broadcasted, e := self.SendETHWithNonceAndPrice(newNonce, priceGwei, ethutils.FloatToBigInt(amount, 18), addr)
		txs = append(txs, tx)
		broadcasteds = append(broadcasteds, broadcasted)
		errors = append(errors, e)
	}
	return txs, broadcasteds, errors
}

func (self *Account) SendETHToMultipleAddresses(amounts []float64, addresses []string) (txs []*types.Transaction, broadcasteds []bool, errors []error) {
	if len(amounts) != len(addresses) {
		panic("amounts and addresses must have the same length")
		return
	}
	nonce, err := self.GetMinedNonce()
	if err != nil {
		panic(fmt.Errorf("cannot get nonce: %s", err))
		return
	}
	priceGwei, err := self.reader.RecommendedGasPrice()
	if err != nil {
		panic(fmt.Errorf("cannot get recommended gas price: %s", err))
	}
	txs = []*types.Transaction{}
	broadcasteds = []bool{}
	errors = []error{}
	for i, addr := range addresses {
		amount := amounts[i]
		newNonce := nonce + uint64(i)
		tx, broadcasted, e := self.SendETHWithNonceAndPrice(newNonce, priceGwei, ethutils.FloatToBigInt(amount, 18), addr)
		txs = append(txs, tx)
		broadcasteds = append(broadcasteds, broadcasted)
		errors = append(errors, e)
	}
	return txs, broadcasteds, errors
}

func (self *Account) CallContractWithPrice(
	priceGwei float64, value float64, caddr string, function string,
	params ...interface{}) (tx *types.Transaction, broadcasted bool, errors error) {
	nonce, err := self.GetMinedNonce()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get nonce: %s", err)
	}
	return self.CallContractWithNonceAndPrice(
		nonce, priceGwei, value, caddr, function, params...)
}

func (self *Account) CallContract(
	value float64, caddr string, function string,
	params ...interface{}) (tx *types.Transaction, broadcasted bool, errors error) {
	nonce, err := self.GetMinedNonce()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get nonce: %s", err)
	}
	priceGwei, err := self.reader.RecommendedGasPrice()
	if err != nil {
		return nil, false, fmt.Errorf("cannot get recommended gas price: %s", err)
	}
	return self.CallContractWithNonceAndPrice(
		nonce, priceGwei, value, caddr, function, params...)
}

func (self *Account) PackData(caddr string, function string, params ...interface{}) ([]byte, error) {
	abi, err := self.reader.GetABI(caddr)
	if err != nil {
		return []byte{}, fmt.Errorf("Cannot get ABI from etherscan for %s", caddr)
	}
	return abi.Pack(function, params...)
}

func (self *Account) CallContractWithNonceAndPrice(
	nonce uint64, priceGwei float64,
	value float64, caddr string, function string,
	params ...interface{}) (tx *types.Transaction, broadcasted bool, errors error) {
	if value < 0 {
		panic("value must be non-negative")
	}
	data, err := self.PackData(caddr, function, params...)
	if err != nil {
		return nil, false, fmt.Errorf("Cannot pack the params: %s", err)
	}
	gasLimit, err := self.reader.EstimateGas(
		self.Address(), caddr, priceGwei, value, data)
	if err != nil {
		return nil, false, fmt.Errorf("Cannot estimate gas: %s", err)
	}
	tx = ethutils.BuildTx(nonce, caddr, value, gasLimit, priceGwei, data)
	signedTx, err := self.signer.SignTx(tx)
	if err != nil {
		return tx, false, fmt.Errorf("couldn't sign the tx: %s", err)
	}
	_, broadcasted, errors = self.broadcaster.BroadcastTx(signedTx)
	return signedTx, broadcasted, errors
}

func NewRopstenAccountFromKeystore(file string, password string) (*Account, error) {
	_, key, err := PrivateKeyFromKeystore(file, password)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewRopstenReader(),
		broadcaster.NewRopstenBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewAccountFromKeystore(file string, password string) (*Account, error) {
	_, key, err := PrivateKeyFromKeystore(file, password)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewEthReader(),
		broadcaster.NewBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewRopstenAccountFromPrivateKeyFile(file string) (*Account, error) {
	_, key, err := PrivateKeyFromFile(file)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewRopstenReader(),
		broadcaster.NewRopstenBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewAccountFromPrivateKeyFile(file string) (*Account, error) {
	_, key, err := PrivateKeyFromFile(file)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewEthReader(),
		broadcaster.NewBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewTrezorAccount(path string, address string) (*Account, error) {
	signer, err := NewTrezorSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewEthReader(),
		broadcaster.NewBroadcaster(),
		common.HexToAddress(address),
	}, nil
}

func NewRopstenTrezorAccount(path string, address string) (*Account, error) {
	signer, err := NewTrezorSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewRopstenReader(),
		broadcaster.NewRopstenBroadcaster(),
		common.HexToAddress(address),
	}, nil
}
