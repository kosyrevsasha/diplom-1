package accrual

import (
	"diplom-1/cmd/gophermart/internal/repository"
)

type Worker struct {
	CheckChanel chan string
}

type Result struct {
	Number string
}

func (w *Worker) Run(db repository.Repository) {
	for number := range w.CheckChanel {
		order := CheckOrder(number)
		if order.Status == repository.PROCESSING {
			w.CheckChanel <- number
			continue
		} else {
			db.UpdateOrder(order)
		}
	}
}
