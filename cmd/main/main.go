package main

import (
	"Intermediate_web3/pkg/database"
	"Intermediate_web3/pkg/service"
	"fmt"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

func main() {
	err := database.Connect()
	if err != nil {
		fmt.Printf("Error connecting to the api")
		return
	}
	defer database.Close()

	//router := gin.Default()
	//err = api.RegisterApi(router)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//err = router.Run()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	err = service.TokenTracking()
	if err != nil {
		return
	}

}
