package tracker

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
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
	return from, to
}

func isUserTracked(address string, chain string) bool {
	trackedUser := mapListTracking[chain].UsersTracking
	if trackedUser == "" {
		return false
	}
	return address == trackedUser
}

func isTokenTracked(address, chain string) bool {
	_, ok := mapListTracking[chain].MapListTokens[address]
	return ok
}
