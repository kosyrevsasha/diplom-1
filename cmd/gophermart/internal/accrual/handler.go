package accrual

import (
	"diplom-1/cmd/gophermart/internal/config"
	"diplom-1/cmd/gophermart/internal/repository"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

var (
	ErrConflict = errors.New("заказ уже в обработке")
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

func RegisterOrder(order Order) error {
	resp := fiber.AcquireResponse()
	o, _ := json.Marshal(order)
	status, _, err := fiber.Post(config.ProcessConfig.Accrual + "/api/orders").
		Body(o).
		ContentType("application/json").
		SetResponse(resp).
		Bytes()
	if status != http.StatusAccepted || len(err) != 0 {
		if status == http.StatusConflict {
			return ErrConflict
		} else {
			return errors.New("ошибка при регистрации заказа")
		}
	}
	return nil
}

func CheckOrder(number string) repository.ProcessedOrder {
	resp := fiber.AcquireResponse()
	statusCode, body, reqErrors := fiber.Get(config.ProcessConfig.Accrual + "/api/orders/" + number).SetResponse(resp).Bytes()
	var err error
	if len(reqErrors) != 0 {
		err = reqErrors[0]
	}
	if statusCode == http.StatusOK && err == nil {
		var order repository.ProcessedOrder
		merr := json.Unmarshal(body, &order)
		if merr != nil {
			return repository.ProcessedOrder{}
		}
		return order
	} else {
		return repository.ProcessedOrder{}
	}
}
