package accrual

import (
	"diplom-1/cmd/gophermart/internal/config"
	"diplom-1/cmd/gophermart/internal/repository"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
)

func RegisterRewards(rewards []Reward) error {
	resp := fiber.AcquireResponse()
	for _, reward := range rewards {
		b, _ := json.Marshal(reward)
		statusCode, _, _ := fiber.Post(config.ProcessConfig.Accrual + "/api/goods").
			Body(b).
			ContentType("application/json").
			SetResponse(resp).
			Bytes()
		if statusCode != http.StatusOK {
			return errors.New("reward registration failed")
		}
	}
	return nil
}

func RegisterOrder(order Order) {
	resp := fiber.AcquireResponse()
	o, _ := json.Marshal(order)
	fiber.Post(config.ProcessConfig.Accrual + "/api/orders").
		Body(o).
		ContentType("application/json").
		SetResponse(resp).
		Bytes()
}

func CheckOrder(number string) repository.ProcessedOrder {
	resp := fiber.AcquireResponse()
	statusCode, body, reqErrors := fiber.Get(config.ProcessConfig.Accrual + "/api/orders/" + number).SetResponse(resp).Bytes()
	for _, er := range reqErrors {
		log.Println("++++ ", er)
	}
	if statusCode == http.StatusOK && err != nil {
		var order repository.ProcessedOrder
		merr := json.Unmarshal(body, &order)
		log.Println(merr)
		if merr != nil {
			return repository.ProcessedOrder{}
		}
		return order
	} else {
		return repository.ProcessedOrder{}
	}
}
