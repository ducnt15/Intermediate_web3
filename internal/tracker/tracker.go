package tracker

import (
	"Intermediate_web3/internal/api"
	token "Intermediate_web3/internal/erc20"
	"Intermediate_web3/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"math/big"
	"os"
	"strconv"
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

func TrackingToken() error {
	if config.Chain == "" {
		return fmt.Errorf("chain configuration not found")
	}
	err := tracker(*config)
	if err != nil {
		return err
	}
	return nil
}

func tracker(chainConfig models.ChainConfig) error {
	client, err := ethclient.Dial(os.Getenv("RPC"))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	chainID, e := client.NetworkID(context.Background())
	if e != nil {
		return fmt.Errorf("failed to get chain ID: %v", e)
	}
	blockNumber := big.NewInt(BlockNumber)
	for {
		block, err := client.BlockByNumber(context.Background(), blockNumber)
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
			// check erc20 token use logs transfer in transactions
			er := trackingErc20Token(client, tx, chainConfig)
			if er != nil {
				fmt.Printf("failed to handle erc20 token tracking info: %v", er)
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
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		fmt.Printf("failed to get transaction receipt: %v", err)
		return err
	}

	for _, log := range receipt.Logs {
		tokenFilterer, er := token.NewStoreFilterer(log.Address, client)
		if er != nil {
			fmt.Printf("failed to create token filterer: %v", err)
			continue
		}
		transfer, e := tokenFilterer.ParseTransfer(*log)
		if e != nil {
			fmt.Printf(e.Error())
			continue
		}
		tokenAddress := strings.ToLower(log.Address.Hex())
		if !isTokenTracked(tokenAddress, chainConfig.Chain) {
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
		amountFloat := new(big.Float).SetInt(amount)
		amountFloat = new(big.Float).Quo(amountFloat, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
		trackingInfo := models.TrackingInformation{
			TransactionHash: tx.Hash().Hex(),
			Type:            TypeTokenERC20,
			From:            fromAddr,
			To:              toAddr,
			Amount:          amountFloat.Text('f', -1),
			Chain:           chainConfig.Chain,
			Symbol:          tokenSymbol,
			Token:           tokenAddress,
		}

		er = notifyAndSaveDB(&trackingInfo, chainConfig)
		if er != nil {
			fmt.Printf("failed to save tracking info: %v", er)
		}
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
		Symbol:          config.ChainSymbol,
		Token:           "",
	}
	fmt.Print(trackingInfo)
	return trackingInfo
}

func checkTransferLog(transfer *token.StoreTransfer, chain string) (string, string, *big.Int, error) {
	fromAddress := strings.ToLower(transfer.From.Hex())
	toAddress := strings.ToLower(transfer.To.Hex())

	if !isUserTracked(fromAddress, chain) && !isUserTracked(toAddress, chain) {
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

	err = sendMessage(message)
	if err != nil {
		return err
	}
	return nil
}

func sendMessage(message string) error {
	TelegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	groupIdStr := os.Getenv("TELEGRAM_CHAT_ID")
	bot, err := tgbotapi.NewBotAPI(TelegramBotToken)
	if err != nil {
		fmt.Println(err)
	}
	groupId, err := strconv.ParseInt(groupIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse GROUPCHAT_ID: %v", err)
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
