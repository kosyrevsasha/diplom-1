package accrual

import (
	"context"
	"diplom-1/cmd/gophermart/internal/repository"
	"time"
)

type Worker struct {
	CheckChanel chan string
}

type Result struct {
	Number string
}

func (w *Worker) Run(ctx context.Context, db repository.Repository) {
	for {
		select {
		case <-ctx.Done():
			return
		case number := <-w.CheckChanel:
			time.Sleep(1 * time.Second)
			order := CheckOrder(number)
			if order.Status == repository.PROCESSING || order.Status == repository.NEW {
				w.CheckChanel <- number
				continue
			} else {
				db.UpdateOrder(order)
			}
		}
	}
}
