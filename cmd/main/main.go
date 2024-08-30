package main

import (
	"Intermediate_web3/internal/config"
	"Intermediate_web3/internal/tracker"
	"context"
	"fmt"
)

func init() {
	err := config.InitConfig()
	if err != nil {
		fmt.Println("Error loading config file")
	}
}

func main() {
	err := tracker.RunTracking(context.Background())
	if err != nil {
		fmt.Printf("Error running tracker: %v", err)
	}
}
