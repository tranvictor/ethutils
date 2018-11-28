package ethutils

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func FloatToInt(amount float64) int64 {
	s := fmt.Sprintf("%.0f", amount)
	if i, err := strconv.Atoi(s); err == nil {
		return int64(i)
	} else {
		panic(err)
	}
}

// FloatToBigInt converts a float to a big int with specific decimal
// Example:
// - FloatToBigInt(1, 4) = 10000
// - FloatToBigInt(1.234, 4) = 12340
func FloatToBigInt(amount float64, decimal int64) *big.Int {
	// 6 is our smallest precision
	if decimal < 6 {
		return big.NewInt(FloatToInt(amount * math.Pow10(int(decimal))))
	}
	result := big.NewInt(FloatToInt(amount * math.Pow10(6)))
	return result.Mul(result, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(decimal-6), nil))
}

// BigToFloat converts a big int to float according to its number of decimal digits
// Example:
// - BigToFloat(1100, 3) = 1.1
// - BigToFloat(1100, 2) = 11
// - BigToFloat(1100, 5) = 0.11
func BigToFloat(b *big.Int, decimal int64) float64 {
	f := new(big.Float).SetInt(b)
	power := new(big.Float).SetInt(new(big.Int).Exp(
		big.NewInt(10), big.NewInt(decimal), nil,
	))
	res := new(big.Float).Quo(f, power)
	result, _ := res.Float64()
	return result
}

// GweiToWei converts Gwei as a float to Wei as a big int
func GweiToWei(n float64) *big.Int {
	return FloatToBigInt(n, 9)
}

// EthToWei converts Gwei as a float to Wei as a big int
func EthToWei(n float64) *big.Int {
	return FloatToBigInt(n, 18)
}

func HexToBig(hex string) *big.Int {
	result, err := hexutil.DecodeBig(hex)
	if err != nil {
		panic(err)
	}
	return result
}

func HexToAddress(hex string) common.Address {
	return common.HexToAddress(hex)
}

func HexToAddresses(hexes []string) []common.Address {
	result := []common.Address{}
	for _, h := range hexes {
		result = append(result, common.HexToAddress(h))
	}
	return result
}
