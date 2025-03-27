package main

import (
	"diplom-1/cmd/gophermart/internal/config"
	"diplom-1/cmd/gophermart/internal/repository"
	"diplom-1/cmd/gophermart/internal/router"
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("---Starting---")

	initErr := config.Init()
	if initErr != nil {
		panic(initErr)
	}

	db, repErr := repository.InitDB()
	if repErr != nil {
		panic(repErr)
	}

	fmt.Println("---Ready---")

	serverErr := http.ListenAndServe(config.ProcessConfig.ServerAddress, router.BuildRouter(db))
	if serverErr != nil {
		panic(serverErr)
	}
}
