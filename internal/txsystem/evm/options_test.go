package evm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WithGasPrice(t *testing.T) {
	const gasPrice = ^uint64(0) // uint64 max value
	opts := &Options{
		gasUnitPrice: big.NewInt(0),
	}
	WithGasPrice(gasPrice)(opts)
	require.Equal(t, gasPrice, opts.gasUnitPrice.Uint64())
}
