package main

import (
	"diplom-1/cmd/gophermart/internal/accrual"
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

	err := accrual.RegisterRewards(accrual.Rewards)
	if err != nil {
		panic(err)
	}

	worker := accrual.Worker{make(chan string)}
	go worker.Run(db)

	serverErr := http.ListenAndServe(config.ProcessConfig.ServerAddress, router.BuildRouter(db, &worker))
	if serverErr != nil {
		panic(serverErr)
	}

}
