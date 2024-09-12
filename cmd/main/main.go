package main

import (
	"Intermediate_web3/internal/api"
	"Intermediate_web3/internal/tracking"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

func main() {
	err := api.Connect()
	if err != nil {
		fmt.Printf("Error connecting to the api")
		return
	}
	defer api.Close()

	router := gin.Default()
	err = api.RegisterApi(router)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = router.Run()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = tracking.TokenTracking()
	if err != nil {
		return
	}

}
