package tracker

import (
	token "Intermediate_web3/internal/erc20"
	"Intermediate_web3/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"io"
	"math/big"
	"os"
	"strconv"
	"time"
)

const (
	TypeTokenNative = "NativeToken"
	TypeTokenERC20  = "Erc20Token"
)

var (
	mapListTracking = make(map[string]struct {
		MapListTokens map[string]bool
		UsersTracking string
	})
	TelegramBotToken string
	groupId          int64
	config           *models.ChainConfig
)

func init() {
	var err error
	err = loadEnv()
	if err != nil {
		panic(err)
	}

	config, err = loadConfig("config.json")
	if err != nil {
		panic(err)
	}
	initConfig()
}

func loadConfig(filePath string) (*models.ChainConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}
	return config, nil
}

func initConfig() {
	mapListTracking[config.Chain] = struct {
		MapListTokens map[string]bool
		UsersTracking string
	}{
		MapListTokens: make(map[string]bool),
		UsersTracking: config.UsersTracking,
	}

	for _, tokenTracking := range config.ListTokensTracking {
		mapListTracking[config.Chain].MapListTokens[tokenTracking] = true
	}
}

func loadEnv() error {
	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	groupIdStr := os.Getenv("TELEGRAM_CHAT_ID")

	groupId, err = strconv.ParseInt(groupIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse GROUPCHAT_ID: %v", err)
	}
	return nil
}

func tracker(chainConfig models.ChainConfig) error {
	client, err := ethclient.Dial(chainConfig.Rpc)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}

	//header, err := client.HeaderByNumber(context.Background(), nil)
	//if err != nil {
	//	return fmt.Errorf("failed to get the latest block header: %v", err)
	//}
	//blockNumber := header.Number
	blockNumber := big.NewInt(20675517)
	for {
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil && err.Error() == ethereum.NotFound.Error() {
			fmt.Printf("Block %v not found yet. Waiting...", blockNumber)
			time.Sleep(12 * time.Second) // Wait
			continue
		}
		if err != nil {
			fmt.Print("Failed to parse ABI:", err)
			return fmt.Errorf("failed to parse ABI: %v", err)
		}

		for _, tx := range block.Transactions() {
			fmt.Printf("Block: %v, Transaction: %v\n", blockNumber, tx.Hash().Hex())
			// check native transfer
			err = trackingNativeToken(tx, chainConfig, chainID)
			if err != nil {
				fmt.Printf("Failed to track native token: %v", err)
			}

			// check stable token use logs transfer in transactions
			er := handleErc20TokenTracking(client, tx, chainConfig)
			if er != nil {
				fmt.Printf("failed to handle stable token tracking info: %v", er)
			}
		}
		blockNumber.Add(blockNumber, big.NewInt(1))
	}
}

func trackingNativeToken(tx *types.Transaction, chainConfig models.ChainConfig, chainID *big.Int) error {
	nativeTransfer := checkNativeToken(tx, chainConfig, chainID)
	if nativeTransfer != nil {
		er := notifyTelegram(*nativeTransfer, chainConfig)
		if er != nil {
			return er
		}
	}
	return nil
}

func handleErc20TokenTracking(client *ethclient.Client, tx *types.Transaction, chainConfig models.ChainConfig) error {
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		fmt.Printf("failed to get transaction receipt: %v", err)
		return err
	}

	for _, vLog := range receipt.Logs {
		tokenFilterer, err := token.NewStoreFilterer(vLog.Address, client)
		if err != nil {
			fmt.Printf("failed to create token filterer: %v", err)
			continue
		}
		transfer, err := tokenFilterer.ParseTransfer(*vLog)
		if err != nil {
			fmt.Printf(err.Error())
			continue
		}

		tokenAddress := vLog.Address.Hex()
		if !isTokenTracked(tokenAddress, chainConfig.Chain) {
			continue
		}

		fromAddr, toAddr, amount, err := checkTransferLog(transfer, chainConfig.Chain)
		if err != nil {
			continue
		}
		decimals := uint8(18)
		if tokenConfig, ok := chainConfig.TrackingTokensConfig[tokenAddress]; ok {
			decimals = tokenConfig.Decimals
		}
		amountFloat := new(big.Float).SetInt(amount)
		amountFloat = new(big.Float).Quo(amountFloat, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
		trackingInfo := models.TrackingInformation{
			TransactionHash: tx.Hash().Hex(),
			Type:            TypeTokenERC20,
			From:            fromAddr,
			To:              toAddr,
			Amount:          amountFloat.Text('f', -1),
			Chain:           chainConfig.Chain,
			Token:           tokenAddress,
		}

		er := notifyTelegram(trackingInfo, chainConfig)
		if er != nil {
			fmt.Printf("failed to save tracking info: %v", er)
		}
	}
	return nil
}

func TrackingToken() error {
	if config.Chain == "" {
		return fmt.Errorf("chain configuration not found")
	}

	chainConfig := models.ChainConfig{
		Chain:                config.Chain,
		Rpc:                  config.Rpc,
		ChainSymbol:          config.ChainSymbol,
		UsersTracking:        config.UsersTracking,
		TrackingTokensConfig: config.TrackingTokensConfig,
		ListTokensTracking:   config.ListTokensTracking,
	}

	if err := tracker(chainConfig); err != nil {
		return err
	}
	return nil
}

func checkNativeToken(tx *types.Transaction, config models.ChainConfig, chainId *big.Int) *models.TrackingInformation {
	from, to := getTransactionAddresses(tx, chainId)

	if !isUserTracked(from, config.Chain) && !isUserTracked(to, config.Chain) {
		return nil
	}

	if tx.Value().Cmp(big.NewInt(0)) < 1 {
		return nil
	}

	realValue := new(big.Float).Quo(new(big.Float).SetInt(tx.Value()), big.NewFloat(1e18))
	trackingInfo := &models.TrackingInformation{
		TransactionHash: tx.Hash().Hex(),
		Type:            TypeTokenNative,
		From:            from,
		To:              to,
		Amount:          realValue.Text('f', -1),
		Chain:           config.Chain,
		Token:           "",
	}
	fmt.Print(trackingInfo)
	return trackingInfo
}
func checkTransferLog(transfer *token.StoreTransfer, chain string) (string, string, *big.Int, error) {
	fromAddress := transfer.From.Hex()
	toAddress := transfer.To.Hex()

	if !isUserTracked(fromAddress, chain) && !isUserTracked(toAddress, chain) {
		return "", "", nil, fmt.Errorf("not tracking this user")
	}
	return fromAddress, toAddress, transfer.Value, nil
}

func notifyTelegram(trackingInfo models.TrackingInformation, chainConfig models.ChainConfig) error {
	tokenSymbol := ""
	switch trackingInfo.Type {
	case TypeTokenNative:
		tokenSymbol = chainConfig.ChainSymbol
	case TypeTokenERC20:
		tokenTrackingConfig, exist := chainConfig.TrackingTokensConfig[trackingInfo.Token]
		if exist {
			tokenSymbol = tokenTrackingConfig.Symbol
		}
	default:
	}
	message := fmt.Sprintf(`Chain: %s
		Transaction: %s
		Transfering %s %s from %s to %s`, trackingInfo.Chain,
		trackingInfo.TransactionHash, trackingInfo.Amount, tokenSymbol, trackingInfo.From, trackingInfo.To)
	err := sendMessage(message)
	if err != nil {
		return err
	}
	return nil
}

func sendMessage(message string) error {

	bot, err := tgbotapi.NewBotAPI(TelegramBotToken)
	if err != nil {
		fmt.Println(err)
	}
	chatId, err := strconv.Atoi(strconv.FormatInt(groupId, 10))
	if err != nil {
		fmt.Println(err)
	}
	// Create a new message to send
	msg := tgbotapi.NewMessage(int64(chatId), message)

	// Send the message
	_, err = bot.Send(msg)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
