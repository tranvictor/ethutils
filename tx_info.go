package ethutils

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TxInfo struct {
	Status  string
	Tx      *Transaction
	Receipt *types.Receipt
}

func (self *TxInfo) GasCost() *big.Int {
	return big.NewInt(0).Mul(
		big.NewInt(int64(self.Receipt.GasUsed)),
		self.Tx.GasPrice(),
	)
}

type Transaction struct {
	*types.Transaction
	Extra TxExtraInfo
}

type TxExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *Transaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.Transaction); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.Extra)
}
