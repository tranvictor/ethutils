package account

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tranvictor/ethutils/account/ledgereum"
	"github.com/tranvictor/ethutils/account/trezoreum"
	"github.com/tranvictor/ethutils/broadcaster"
	"github.com/tranvictor/ethutils/reader"
)

func NewRinkebyAccountFromKeystore(file string, password string) (*Account, error) {
	_, key, err := PrivateKeyFromKeystore(file, password)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewRinkebyReader(),
		broadcaster.NewRinkebyBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewRinkebyAccountFromPrivateKey(hex string) (*Account, error) {
	_, key, err := PrivateKeyFromHex(hex)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewRinkebyReader(),
		broadcaster.NewRinkebyBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewRinkebyAccountFromPrivateKeyFile(file string) (*Account, error) {
	_, key, err := PrivateKeyFromFile(file)
	if err != nil {
		return nil, err
	}
	return &Account{
		NewKeySigner(key),
		reader.NewRinkebyReader(),
		broadcaster.NewRinkebyBroadcaster(),
		crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func NewRinkebyTrezorAccount(path string, address string) (*Account, error) {
	signer, err := trezoreum.NewTrezorSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewRinkebyReader(),
		broadcaster.NewRinkebyBroadcaster(),
		common.HexToAddress(address),
	}, nil
}

func NewRinkebyLedgerAccount(path string, address string) (*Account, error) {
	signer, err := ledgereum.NewLedgerSigner(path, address)
	if err != nil {
		return nil, err
	}
	return &Account{
		signer,
		reader.NewRinkebyReader(),
		broadcaster.NewRinkebyBroadcaster(),
		common.HexToAddress(address),
	}, nil
}
