package service

import (
	"Intermediate_web3/internal/api"
	token "Intermediate_web3/internal/build"
	"Intermediate_web3/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	TypeTokenNative = "NativeToken"
	TypeTokenERC20  = "Erc20Token"
	BlockNumber     = 20688778
)

var (
	mapListTracking = make(map[string]struct {
		MapListTokens map[string]bool
		UsersTracking string
	})
	config *models.ChainConfig
	ctx    = context.Background()
)

func init() {
	var err error
	config, err = loadConfig()
	if err != nil {
		panic(err)
	}
	mapListTracking[config.Chain] = struct {
		MapListTokens map[string]bool
		UsersTracking string
	}{
		MapListTokens: make(map[string]bool),
		UsersTracking: config.UsersTracking,
	}

	for _, tokenTracking := range config.ListTokensTracking {
		mapListTracking[config.Chain].MapListTokens[strings.ToLower(tokenTracking)] = true
	}
}

func loadConfig() (*models.ChainConfig, error) {
	byteValue, err := os.ReadFile("config.json")
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}
	return config, nil
}

func TokenTracking() error {
	if config.Chain == "" {
		return fmt.Errorf("chain configuration not found")
	}
	err := handlerTracking(*config)
	if err != nil {
		return err
	}
	return nil
}

func handlerTracking(chainConfig models.ChainConfig) error {
	client, err := ethclient.Dial(os.Getenv("RPC"))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	chainID, e := client.NetworkID(ctx)
	if e != nil {
		return fmt.Errorf("failed to get chain ID: %v", e)
	}
	blockNumber := big.NewInt(BlockNumber)
	for {
		block, err := client.BlockByNumber(ctx, blockNumber)
		if err != nil && err.Error() == ethereum.NotFound.Error() {
			fmt.Printf("Block %v not found yet", blockNumber)
			time.Sleep(12 * time.Second)
			continue
		}
		if err != nil {
			fmt.Print("Failed to parse ABI:", err)
			return fmt.Errorf("failed to parse ABI: %v", err)
		}

		for _, tx := range block.Transactions() {
			fmt.Printf("Block: %v\n", blockNumber)
			// check native transfer
			err = trackingNativeToken(tx, chainConfig, chainID)
			if err != nil {
				fmt.Printf("Failed to track native token: %v", err)
			}
			// check Erc20 token transfer
			err = trackingErc20Token(client, tx, chainConfig)
			if err != nil {
				fmt.Printf("failed to handle build token tracking info: %v", err)
			}
		}
		blockNumber.Add(blockNumber, big.NewInt(1))
	}
}

func trackingNativeToken(tx *types.Transaction, chainConfig models.ChainConfig, chainID *big.Int) error {
	nativeTransfer := checkNativeToken(tx, chainConfig, chainID)
	if nativeTransfer != nil {
		err := notifyAndSaveDB(nativeTransfer, chainConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func trackingErc20Token(client *ethclient.Client, tx *types.Transaction, chainConfig models.ChainConfig) error {
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		fmt.Printf("failed to get transaction receipt: %v", err)
		return err
	}

	for _, log := range receipt.Logs {
		tokenFilterer, err := token.NewStoreFilterer(log.Address, client)
		if err != nil {
			fmt.Printf("failed to create token filterer: %v", err)
			continue
		}
		transfer, err := tokenFilterer.ParseTransfer(*log)
		if err != nil {
			fmt.Printf(err.Error())
			continue
		}
		tokenAddress := strings.ToLower(log.Address.Hex())
		_, ok := mapListTracking[chainConfig.Chain].MapListTokens[tokenAddress]
		if !ok {
			continue
		}
		fromAddr, toAddr, amount, err := checkTransferLog(transfer, chainConfig.Chain)
		if err != nil {
			fmt.Printf("failed to check transfer log: %v", err)
			continue
		}

		tokenContract, err := token.NewStore(log.Address, client)
		if err != nil {
			fmt.Printf("failed to create token contract: %v", err)
			continue
		}
		tokenSymbol, err := tokenContract.Symbol(nil)
		if err != nil {
			fmt.Printf("failed to get token symbol: %v", err)
			continue
		}
		decimals := uint8(18)
		tokenConfig, ok := chainConfig.TrackingTokensConfig[tokenAddress]
		if ok {
			decimals = tokenConfig.Decimals
		}
		amountTransfer := new(big.Float).SetInt(amount)
		amountTransfer = new(big.Float).Quo(amountTransfer, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
		trackingInfo := models.TrackingInformation{
			TransactionHash: tx.Hash().Hex(),
			Type:            TypeTokenERC20,
			From:            fromAddr,
			To:              toAddr,
			Amount:          amountTransfer.Text('f', -1),
			Chain:           chainConfig.Chain,
			Symbol:          tokenSymbol,
			Token:           tokenAddress,
		}
		err = notifyAndSaveDB(&trackingInfo, chainConfig)
		if err != nil {
			fmt.Printf("failed to save tracking info: %v", err)
		}
	}
	return nil
}

func checkNativeToken(tx *types.Transaction, config models.ChainConfig, chainId *big.Int) *models.TrackingInformation {
	from, to := getTransactionAddresses(tx, chainId)

	if !checkUserTracked(from, config.Chain) && !checkUserTracked(to, config.Chain) {
		return nil
	}

	if tx.Value().Cmp(big.NewInt(0)) < 1 {
		return nil
	}

	value := new(big.Float).Quo(new(big.Float).SetInt(tx.Value()), big.NewFloat(1e18))
	trackingInfo := &models.TrackingInformation{
		TransactionHash: tx.Hash().Hex(),
		Type:            TypeTokenNative,
		From:            from,
		To:              to,
		Amount:          value.Text('f', -1),
		Chain:           config.Chain,
		Symbol:          config.ChainSymbol,
		Token:           "",
	}
	fmt.Print(trackingInfo)
	return trackingInfo
}

func checkTransferLog(transfer *token.StoreTransfer, chain string) (string, string, *big.Int, error) {
	fromAddress := strings.ToLower(transfer.From.Hex())
	toAddress := strings.ToLower(transfer.To.Hex())

	if !checkUserTracked(fromAddress, chain) && !checkUserTracked(toAddress, chain) {
		return "", "", nil, fmt.Errorf("not tracking this user")
	}
	return fromAddress, toAddress, transfer.Value, nil
}

func notifyAndSaveDB(trackingInfo *models.TrackingInformation, chainConfig models.ChainConfig) error {
	tokenSymbol := ""
	switch trackingInfo.Type {
	case TypeTokenNative:
		tokenSymbol = chainConfig.ChainSymbol
	case TypeTokenERC20:
		tokenTrackingConfig, ok := chainConfig.TrackingTokensConfig[trackingInfo.Token]
		if ok {
			tokenSymbol = tokenTrackingConfig.Symbol
		}
	default:
	}

	err := api.SaveDB(trackingInfo)
	if err != nil {
		fmt.Printf("failed to save tracking info: %v", err)
	}

	message := fmt.Sprintf(`Chain: %s
			Transaction: %s
			Transfering %s %s
			From %s to %s`, trackingInfo.Chain,
		trackingInfo.TransactionHash, trackingInfo.Amount, tokenSymbol, trackingInfo.From, trackingInfo.To)

	err = SendMessage(message)
	if err != nil {
		return err
	}
	return nil
}
func getTransactionAddresses(tx *types.Transaction, chainID *big.Int) (string, string) {
	from, to := "", ""
	sender, err := types.Sender(types.NewLondonSigner(chainID), tx)
	if err == nil {
		from = sender.Hex()
	}
	to = tx.To().Hex()
	return strings.ToLower(from), strings.ToLower(to)
}

func checkUserTracked(address string, chain string) bool {
	address = strings.ToLower(address)
	trackedUser := mapListTracking[chain].UsersTracking
	if trackedUser == "" {
		return false
	}
	return address == strings.ToLower(trackedUser)
}
