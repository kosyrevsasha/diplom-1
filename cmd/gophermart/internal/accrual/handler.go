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
		log.Printf("Goods: %s - %d", reward.Match, statusCode)
		if statusCode != http.StatusOK {
			return errors.New("Reward registration failed")
		}
	}
	return nil
}

func RegisterOrder(order Order) {
	resp := fiber.AcquireResponse()
	o, _ := json.Marshal(order)
	statusCode, _, _ := fiber.Post(config.ProcessConfig.Accrual + "/api/orders").
		Body(o).
		ContentType("application/json").
		SetResponse(resp).
		Bytes()
	log.Printf("Orders: %s - %d", order.Number, statusCode)
}

func CheckOrder(number string) repository.ProcessedOrder {
	log.Printf("REQUEST !!! %s", number)
	resp := fiber.AcquireResponse()
	statusCode, body, _ := fiber.Get(config.ProcessConfig.Accrual + "/api/orders/" + number).SetResponse(resp).Bytes()
	log.Printf("Order reward: code - %d", statusCode)
	if statusCode == http.StatusOK {
		var order repository.ProcessedOrder
		json.Unmarshal(body, &order)
		return order
	} else {
		return repository.ProcessedOrder{}
	}
}
