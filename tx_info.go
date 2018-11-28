package ethutils

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TxInfo struct {
	Status  string
	Tx      *Transaction
	Receipt *types.Receipt
}

type Transaction struct {
	*types.Transaction
	Extra txExtraInfo
}

type txExtraInfo struct {
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
