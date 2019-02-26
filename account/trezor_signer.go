package account

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/tranvictor/trezoreum"
)

type TrezorSigner struct {
	path           accounts.DerivationPath
	mu             sync.Mutex
	devmu          sync.Mutex
	deviceUnlocked bool
	trezor         trezoreum.Bridge
}

func (self *TrezorSigner) Unlock() error {
	self.devmu.Lock()
	defer self.devmu.Unlock()
	info, state, err := self.trezor.Init()
	if err != nil {
		return err
	}
	fmt.Printf("Firmware version: %d.%d.%d\n", *info.MajorVersion, *info.MinorVersion, *info.PatchVersion)
	for state != trezoreum.Ready {
		if state == trezoreum.WaitingForPin {
			pin := trezoreum.PromptPINFromStdin()
			state, err = self.trezor.UnlockByPin(pin)
			if err != nil {
				fmt.Printf("Pin error: %s\n", err)
			}
		} else if state == trezoreum.WaitingForPassphrase {
			fmt.Printf("Not support passphrase yet\n")
		}
	}
	self.deviceUnlocked = true
	return nil
}

func (self *TrezorSigner) SignTx(tx *types.Transaction) (*types.Transaction, error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	fmt.Printf("Going to proceed signing procedure\n")
	var err error
	if !self.deviceUnlocked {
		err = self.Unlock()
		if err != nil {
			return tx, err
		}
	}
	_, tx, err = self.trezor.Sign(self.path, tx, big.NewInt(1))
	return tx, err
}

func NewRopstenTrezorSigner(path string, address string) (*TrezorSigner, error) {
	p, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	trezor, err := trezoreum.NewTrezoreum()
	if err != nil {
		return nil, err
	}
	return &TrezorSigner{
		p,
		sync.Mutex{},
		sync.Mutex{},
		false,
		trezor,
	}, nil
}

func NewTrezorSigner(path string, address string) (*TrezorSigner, error) {
	p, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	trezor, err := trezoreum.NewTrezoreum()
	if err != nil {
		return nil, err
	}
	return &TrezorSigner{
		p,
		sync.Mutex{},
		sync.Mutex{},
		false,
		trezor,
	}, nil
}
