package reader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	ETHERSCAN_DOMAIN         string = "api.etherscan.io"
	RINKEBY_ETHERSCAN_DOMAIN string = "api-rinkeby.etherscan.io"
	KOVAN_ETHERSCAN_DOMAIN   string = "api-kovan.etherscan.io"
	ROPSTEN_ETHERSCAN_DOMAIN string = "api-ropsten.etherscan.io"

	BSCSCAN_DOMAIN         string = "api.bscscan.com"
	TESTNET_BSCSCAN_DOMAIN string = "api-testnet.bscscan.com"
)

// gas station response
type abiresponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func (ar *abiresponse) IsOK() bool {
	return ar.Status == "1"
}

func (self *EthReader) GetABIFromEtherscan(address string, domain string, apikey string) (*abi.ABI, error) {
	body, err := self.GetABIStringFromEtherscan(address, domain, apikey)
	if err != nil {
		return nil, err
	}
	result, err := abi.JSON(strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (self *EthReader) GetABIStringFromEtherscan(address string, domain string, apikey string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://%s/api?module=contract&action=getabi&address=%s&apikey=%s", domain, address, apikey))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	abiresp := abiresponse{}
	err = json.Unmarshal(body, &abiresp)
	if err != nil {
		return "", err
	}
	if !abiresp.IsOK() {
		return "", fmt.Errorf("error from %s: %s", domain, abiresp.Message)
	}
	return abiresp.Result, err
}

func (self *EthReader) GetEthereumABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetEthereumABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetBSCTestnetABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, TESTNET_BSCSCAN_DOMAIN, self.bscscanAPIKey)
}

func (self *EthReader) GetBSCTestnetABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, TESTNET_BSCSCAN_DOMAIN, self.bscscanAPIKey)
}

func (self *EthReader) GetBSCABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, BSCSCAN_DOMAIN, self.bscscanAPIKey)
}

func (self *EthReader) GetBSCABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, BSCSCAN_DOMAIN, self.bscscanAPIKey)
}

func (self *EthReader) GetRinkebyABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, RINKEBY_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetRinkebyABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, RINKEBY_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetKovanABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, KOVAN_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetKovanABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, KOVAN_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetRopstenABIString(address string) (string, error) {
	return self.GetABIStringFromEtherscan(address, ROPSTEN_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

func (self *EthReader) GetRopstenABI(address string) (*abi.ABI, error) {
	return self.GetABIFromEtherscan(address, ROPSTEN_ETHERSCAN_DOMAIN, self.etherscanAPIKey)
}

// gas station response
type tomoabiresponse struct {
	Contract struct {
		ABICode string `json:"abiCode"`
	} `json:"contract"`
}

func (self *EthReader) GetTomoABIString(address string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://scan.tomochain.com/api/accounts/%s", address))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	abiresp := tomoabiresponse{}
	err = json.Unmarshal(body, &abiresp)
	if err != nil {
		return "", err
	}
	return abiresp.Contract.ABICode, nil
}

func (self *EthReader) GetTomoABI(address string) (*abi.ABI, error) {
	body, err := self.GetTomoABIString(address)
	if err != nil {
		return nil, err
	}
	result, err := abi.JSON(strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (self *EthReader) GetABIString(address string) (string, error) {
	switch self.chain {
	case "ethereum":
		return self.GetEthereumABIString(address)
	case "ropsten":
		return self.GetRopstenABIString(address)
	case "kovan":
		return self.GetKovanABIString(address)
	case "rinkeby":
		return self.GetRinkebyABIString(address)
	case "tomo":
		return self.GetTomoABIString(address)
	case "bsc":
		return self.GetBSCABIString(address)
	case "bsc-test":
		return self.GetBSCTestnetABIString(address)
	}
	return "", fmt.Errorf("'%s' chain is not supported", self.chain)
}

func (self *EthReader) GetABI(address string) (*abi.ABI, error) {
	switch self.chain {
	case "ethereum":
		return self.GetEthereumABI(address)
	case "ropsten":
		return self.GetRopstenABI(address)
	case "kovan":
		return self.GetKovanABI(address)
	case "rinkeby":
		return self.GetRinkebyABI(address)
	case "tomo":
		return self.GetTomoABI(address)
	case "bsc":
		return self.GetBSCABI(address)
	case "bsc-test":
		return self.GetBSCTestnetABI(address)
	}
	return nil, fmt.Errorf("'%s' chain is not supported", self.chain)
}
