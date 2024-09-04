package models

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
	TransactionHash string `json:"transactionHash"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	Chain           string `json:"chain"`
	Token           string `json:"token"`
	Amount          string `json:"amount"`
}
