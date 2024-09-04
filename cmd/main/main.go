package main

import (
	"Intermediate_web3/internal/tracker"
	"fmt"
)

func main() {
	err := tracker.TrackingToken()
	if err != nil {
		fmt.Printf("Error running tracker: %v", err)
	}
}
