package models

import "github.com/uptrace/bun"

type TokenConfig struct {
	TokenName string `json:"TokenName"`
	Symbol    string `json:"Symbol"`
	Decimals  uint8  `json:"Decimals"`
}

type ChainConfig struct {
	Chain                string                 `json:"chain"`
	Rpc                  string                 `json:"rpc"`
	ChainSymbol          string                 `json:"chainSymbol"`
	UsersTracking        string                 `json:"usersTracking"`
	TrackingTokensConfig map[string]TokenConfig `json:"trackingTokensConfig"`
	ListTokensTracking   []string               `json:"listTokensTracking"`
}

type TrackingInformation struct {
	bun.BaseModel   `bun:"table:tracking"`
	ID              int    `bun:",pk,autoincrement"`
	TransactionHash string `bun:"transactionHash,notnull" json:"transactionHash"`
	Type            string `bun:"type,notnull" json:"type"`
	From            string `bun:"from,notnull" json:"from"`
	To              string `bun:"to,notnull" json:"to"`
	Chain           string `bun:"chain,notnull" json:"chain"`
	Token           string `bun:"token" json:"token"`
	Amount          string `bun:"amount,notnull" json:"amount"`
}
