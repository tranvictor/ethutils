package account

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type KeySigner struct {
	key *ecdsa.PrivateKey
}

func (self *KeySigner) SignTx(tx *types.Transaction) (*types.Transaction, error) {
	opts := bind.NewKeyedTransactor(self.key)
	return opts.Signer(types.HomesteadSigner{}, crypto.PubkeyToAddress(self.key.PublicKey), tx)
}

func NewKeySigner(key *ecdsa.PrivateKey) *KeySigner {
	return &KeySigner{key}
}
