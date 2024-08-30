package tracker

import (
	"Intermediate_web3/internal/config"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"sync"
	"time"
)

func RunTracking(ctx context.Context) error {
	var wg sync.WaitGroup

	chainConfigs := []config.ChainConfig{config.Config}

	for _, chainConfig := range chainConfigs {
		wg.Add(1)
		go func(chainConfig config.ChainConfig) {
			defer wg.Done()
			client, err := ethclient.Dial(chainConfig.Gpc)
			if err != nil {
				fmt.Printf("Failed to connect to Ethereum client: %v\n", err)
				return
			}
			defer client.Close()
			err = startTracking(ctx, client, chainConfig)
			if err != nil {
				fmt.Printf("Failed to start tracking for chain %s: %v", chainConfig.Chain, err)
			}
		}(chainConfig)
	}
	wg.Wait()

	return nil
}

func startTracking(ctx context.Context, client *ethclient.Client, chainConfig config.ChainConfig) error {
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to get the latest block header: %v", err)
	}
	blockNumber := header.Number // Start block
	for {
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil && err.Error() == ethereum.NotFound.Error() {
			time.Sleep(10 * time.Second) // Wait
			continue
		}

		if err != nil {
			return fmt.Errorf("failed to parse ABI: %v", err)
		}

		for _, tx := range block.Transactions() {
			fmt.Printf("Block: %v , Transaction: %v", blockNumber, tx.Hash().Hex())
			// check native token transfer in transactions
			err = trackingNativeToken(tx, chainConfig, chainID, ctx)
			if err != nil {
				fmt.Printf("Failed to track native token: %v", err)
			}

			// check erc20 token use logs transfer in transactions
			er := handleErc20TokenTracking(client, tx, chainConfig, ctx)
			if er != nil {
				fmt.Printf("failed to handle erc20 token tracking info: %v", er)
			}
		}
		blockNumber.Add(blockNumber, big.NewInt(1))
	}
	return nil
}

func handleErc20TokenTracking(client *ethclient.Client, tx *types.Transaction, chainConfig config.ChainConfig, ctx context.Context) interface{} {
	return nil
}

func trackingNativeToken(tx *types.Transaction, chainConfig config.ChainConfig, chainID *big.Int, ctx context.Context) error {
	if tx.Value().Cmp(big.NewInt(0)) > 0 {
		fromAddress, err := types.Sender(types.NewEIP155Signer(chainID), tx)
		if err != nil {
			return fmt.Errorf("failed to get sender address: %v", err)
		}

		toAddress := tx.To()
		value := tx.Value()

		message := fmt.Sprintf("Native token transfer detected:\nFrom: %s\n", fromAddress.Hex())
		if toAddress != nil {
			message += fmt.Sprintf("To: %s\n", toAddress.Hex())
		} else {
			message += "To: Contract creation\n"
		}
		message += fmt.Sprintf("Value: %s\nChain: %s", value.String(), chainConfig.Chain)

		// Send message via Telegram
		err = sendTelegramMessage(message)
		if err != nil {
			return fmt.Errorf("failed to send Telegram message: %v", err)
		}
	}
	return nil
}

func sendTelegramMessage(message string) error {

	return nil
}
