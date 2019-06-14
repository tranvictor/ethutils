package account

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tranvictor/ethutils/account/ledgereum"
	"github.com/tranvictor/ethutils/account/trezor"
	"github.com/tranvictor/ethutils/broadcaster"
	"github.com/tranvictor/ethutils/reader"
)

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

func NewAccountFromPrivateKey(hex string) (*Account, error) {
	_, key, err := PrivateKeyFromHex(hex)
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

func NewLedgerAccount(path string, address string) (*Account, error) {
	signer, err := ledgereum.NewLedgerSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		// nil,
		reader.NewEthReader(),
		broadcaster.NewBroadcaster(),
		common.HexToAddress(address),
	}, nil
}

func NewTrezorAccount(path string, address string) (*Account, error) {
	signer, err := trezor.NewTrezorSigner(path, address)
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
