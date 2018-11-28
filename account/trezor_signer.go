package account

import (
	"fmt"
	"math/big"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/ssh/terminal"
)

type TrezorSigner struct {
	path           accounts.DerivationPath
	mu             sync.Mutex
	wallet         accounts.Wallet
	cwallet        chan accounts.Wallet
	monitorRunning bool
}

func getPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	return string(bytePassword)
}

func (self *TrezorSigner) promptPassword() string {
	return getPassword(
		"Pin required to open Trezor wallet\n" +
			"Look at the device for number positions\n\n" +
			"7 | 8 | 9\n" +
			"--+---+--\n" +
			"4 | 5 | 6\n" +
			"--+---+--\n" +
			"1 | 2 | 3\n\n" +
			"Enter your PIN: ",
	)
}

func (self *TrezorSigner) monitor(channel chan accounts.WalletEvent) {
	for event := range channel {
		switch event.Kind {
		case accounts.WalletArrived:
			self.cwallet <- event.Wallet
		case accounts.WalletDropped:
			if self.wallet != nil {
				self.wallet.Close()
				self.wallet = nil
			}
		}
	}
}

func (self *TrezorSigner) ConnectAndMonitorTrezor() error {
	trezorHub, err := usbwallet.NewTrezorHub()
	if err != nil {
		return err
	}
	if len(trezorHub.Wallets()) > 0 {
		self.wallet = trezorHub.Wallets()[0]
	}
	channel := make(chan accounts.WalletEvent)
	trezorHub.Subscribe(channel)
	go self.monitor(channel)
	self.monitorRunning = true
	return nil
}

func (self *TrezorSigner) GetWallet() accounts.Wallet {
	if self.wallet == nil {
		fmt.Printf("Please connect your Trezor...\n")
		self.wallet = <-self.cwallet
	}
	return self.wallet
}

func (self *TrezorSigner) SignTx(tx *types.Transaction) (*types.Transaction, error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	if !self.monitorRunning {
		err := self.ConnectAndMonitorTrezor()
		if err != nil {
			fmt.Printf("Trying to connect to trezor failed: %s\n", err)
		}
	}
	acc, err := self.GetWallet().Derive(self.path, true)
	if err != nil {
		if err = self.GetWallet().Open(""); err != nil {
			if err == usbwallet.ErrTrezorPINNeeded {
				password := self.promptPassword()
				err = self.GetWallet().Open(password)
			}
		}
		if err != nil {
			return nil, err
		} else {
			acc, err = self.GetWallet().Derive(self.path, true)
			if err != nil {
				return nil, err
			} else {
				fmt.Printf("\naccount is unlocked. Please approve on your Trezor to sign...\n")
			}
		}
	}
	signedTx, err := self.GetWallet().SignTx(acc, tx, big.NewInt(1))
	if err != nil {
		fmt.Printf("Couldn't sign tx: %s\n", err)
	} else {
		fmt.Printf("Tx is signed successfully.\n")
	}
	return signedTx, err
}

func NewRopstenTrezorSigner(path string, address string) (*TrezorSigner, error) {
	p, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	return &TrezorSigner{
		p,
		sync.Mutex{},
		nil,
		make(chan accounts.Wallet),
		false,
	}, nil
}

func NewTrezorSigner(path string, address string) (*TrezorSigner, error) {
	p, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	return &TrezorSigner{
		p,
		sync.Mutex{},
		nil,
		make(chan accounts.Wallet),
		false,
	}, nil
}
