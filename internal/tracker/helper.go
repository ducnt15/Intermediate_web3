package tracker

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

func getTransactionAddresses(tx *types.Transaction, chainID *big.Int) (string, string) {
	from, to := "", ""
	sender, err := types.Sender(types.NewLondonSigner(chainID), tx)
	if err == nil {
		from = sender.Hex()
	}
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	return strings.ToLower(from), strings.ToLower(to)
}

func isUserTracked(address string, chain string) bool {
	address = strings.ToLower(address)
	trackedUser := mapListTracking[chain].UsersTracking
	if trackedUser == "" {
		return false
	}
	return address == strings.ToLower(trackedUser)
}

func isTokenTracked(address, chain string) bool {
	_, ok := mapListTracking[chain].MapListTokens[address]
	return ok
}
