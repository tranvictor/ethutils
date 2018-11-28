package txanalyzer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/user"
	"path"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
)

type DefaultAddressDatabase struct {
	data map[common.Address]string
}

func (self *DefaultAddressDatabase) Register(addr string, name string) {
	self.data[common.HexToAddress(addr)] = name
}

func (self *DefaultAddressDatabase) GetName(addr string) string {
	name, found := self.data[common.HexToAddress(addr)]
	if found {
		return name
	} else {
		return "unknown"
	}
}

type tokens []struct {
	Address string `json:"address"`
	Symbol  string `json:"symbol"`
}

func registerFromFile(filename string, db *DefaultAddressDatabase) error {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("couldn't get filepath of the caller")
	}
	content, err := ioutil.ReadFile(path.Join(path.Dir(current), filename))
	if err != nil {
		return err
	}
	ts := tokens{}
	err = json.Unmarshal(content, &ts)
	if err != nil {
		return err
	}
	for _, t := range ts {
		db.Register(t.Address, t.Symbol)
	}
	return nil
}

func getDataFromDefaultFile() map[string]string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	content, err := ioutil.ReadFile(path.Join(dir, "addresses.json"))
	if err != nil {
		fmt.Printf("reading addresses from ~/addresses.json failed: %s. Ignored.\n", err)
		return map[string]string{}
	}
	result := map[string]string{}
	err = json.Unmarshal(content, &result)
	if err != nil {
		fmt.Printf("reading addresses from ~/addresses.json failed: %s. Ignored.\n", err)
		return map[string]string{}
	}
	return result
}

func NewDefaultAddressDatabase() *DefaultAddressDatabase {

	// get data from ~/addresses.json, expecting a map
	// from address (string) to name (string)
	data := getDataFromDefaultFile()

	db := &DefaultAddressDatabase{
		data: map[common.Address]string{},
	}

	for addr, name := range data {
		db.Register(addr, name)
	}

	err := registerFromFile("tokens.json", db)
	if err != nil {
		fmt.Printf("Loading token addresses from file failed: %s\n", err)
	}

	return db
}
