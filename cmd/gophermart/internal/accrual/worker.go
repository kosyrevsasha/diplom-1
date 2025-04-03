package accrual

import (
	"diplom-1/cmd/gophermart/internal/repository"
	"fmt"
)

type Worker struct {
	CheckChanel chan string
}

type Result struct {
	Number string
}

func (w *Worker) Run(db repository.Repository) {
	for number := range w.CheckChanel {
		fmt.Printf("____%s___", number)
		order := CheckOrder(number)
		if order.Status == repository.PROCESSING {
			w.CheckChanel <- number
			continue
		} else {
			db.UpdateOrder(order)
		}
	}
}
