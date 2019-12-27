package ledgereum

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	kusb "github.com/karalabe/usb"
)

const (
	LEDGER_VENDOR_ID   uint16 = 0x2c97
	LEDGER_USAGE_ID    uint16 = 0xffa0
	LEDGER_ENDPOINT_ID int    = 0
)

var LEDGER_PRODUCT_IDS []uint16 = []uint16{
	// Original product IDs
	0x0000, /* Ledger Blue */
	0x0001, /* Ledger Nano S */
	0x0004, /* Ledger Nano X */

	// Upcoming product IDs: https://www.ledger.com/2019/05/17/windows-10-update-sunsetting-u2f-tunnel-transport-for-ledger-devices/
	0x0015, /* HID + U2F + WebUSB Ledger Blue */
	0x1015, /* HID + U2F + WebUSB Ledger Nano S */
	0x4015, /* HID + U2F + WebUSB Ledger Nano X */
	0x0011, /* HID + WebUSB Ledger Blue */
	0x1011, /* HID + WebUSB Ledger Nano S */
	0x4011, /* HID + WebUSB Ledger Nano X */
}

type LedgerSigner struct {
	path           accounts.DerivationPath
	driver         *ledgerDriver
	device         kusb.Device
	deviceUnlocked bool
	mu             sync.Mutex
	devmu          sync.Mutex
	chainID        int64
}

func (self *LedgerSigner) Unlock() error {
	self.devmu.Lock()
	defer self.devmu.Unlock()
	infos, err := kusb.Enumerate(LEDGER_VENDOR_ID, 0)
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		return fmt.Errorf("Ledger device is not found")
	} else {
		for _, info := range infos {
			for _, id := range LEDGER_PRODUCT_IDS {
				// Windows and Macos use UsageID matching, Linux uses Interface matching
				if info.ProductID == id && (info.UsagePage == LEDGER_USAGE_ID || info.Interface == LEDGER_ENDPOINT_ID) {
					self.device, err = info.Open()
					if err != nil {
						return err
					}
					if err = self.driver.Open(self.device, ""); err != nil {
						return err
					}
					break
				}
			}
		}
	}
	self.deviceUnlocked = true
	return nil
}

func (self *LedgerSigner) SignTx(tx *types.Transaction) (*types.Transaction, error) {
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
	_, tx, err = self.driver.ledgerSign(self.path, tx, big.NewInt(self.chainID))
	return tx, err
}

func NewLedgerSigner(path string, address string) (*LedgerSigner, error) {
	p, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	return &LedgerSigner{
		p,
		newLedgerDriver(),
		nil,
		false,
		sync.Mutex{},
		sync.Mutex{},
		1,
	}, nil
}
