package reader

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	eu "github.com/tranvictor/ethutils"
)

type EthereumNode interface {
	NodeName() string
	NodeURL() string
	EstimateGas(from, to string, priceGwei float64, value *big.Int, data []byte) (gas uint64, err error)
	GetCode(address string) (code []byte, err error)
	GetBalance(address string) (balance *big.Int, err error)
	GetMinedNonce(address string) (nonce uint64, err error)
	GetPendingNonce(address string) (nonce uint64, err error)
	TransactionReceipt(txHash string) (receipt *types.Receipt, err error)
	TransactionByHash(txHash string) (tx *eu.Transaction, isPending bool, err error)
	// Call(result interface{}, method string, args ...interface{}) error
	GetGasPriceSuggestion() (*big.Int, error)
	ReadContractToBytes(atBlock int64, from string, caddr string, abi *abi.ABI, method string, args ...interface{}) ([]byte, error)
	HeaderByNumber(number int64) (*types.Header, error)
	GetLogs(fromBlock, toBlock int, addresses []string, topic string) ([]types.Log, error)
	CurrentBlock() (uint64, error)
}
