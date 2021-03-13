package account

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tranvictor/ethutils/account/ledgereum"
	"github.com/tranvictor/ethutils/account/trezoreum"
	"github.com/tranvictor/ethutils/broadcaster"
	"github.com/tranvictor/ethutils/reader"
)

func NewBSCAccountFromKeystore(file string, password string) (*Account, error) {
	_, key, err := PrivateKeyFromKeystore(file, password)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewBSCReader(),
		broadcaster.NewBSCBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewBSCAccountFromPrivateKey(hex string) (*Account, error) {
	_, key, err := PrivateKeyFromHex(hex)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewBSCReader(),
		broadcaster.NewBSCBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewBSCAccountFromPrivateKeyFile(file string) (*Account, error) {
	_, key, err := PrivateKeyFromFile(file)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewBSCReader(),
		broadcaster.NewBSCBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewBSCTrezorAccount(path string, address string) (*Account, error) {
	signer, err := trezoreum.NewTrezorSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewBSCReader(),
		broadcaster.NewBSCBroadcaster(),
		common.HexToAddress(address),
	}, nil
}

func NewBSCLedgerAccount(path string, address string) (*Account, error) {
	signer, err := ledgereum.NewLedgerSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewBSCReader(),
		broadcaster.NewBSCBroadcaster(),
		common.HexToAddress(address),
	}, nil
}
