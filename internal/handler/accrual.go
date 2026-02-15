package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mdflamingo/Gofermart/internal/repository"
)


func StartAccrualWorker(ctx context.Context, storage *repository.DBStorage) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := storage.GetOrdersToUpdate()
			if err != nil {
				continue
			}

			for _, order := range orders {
				resp, err := accrualClient.R().Get("/api/orders/" + order.Number)
				if err != nil || resp.StatusCode() != 200 {
					continue
				}

				var accrualResp struct {
					Status  string  `json:"status"`
					Accrual float64 `json:"accrual"`
				}
				json.Unmarshal(resp.Body(), &accrualResp)

				err = storage.UpdateOrderStatus(order.ID, accrualResp.Status, accrualResp.Accrual)
				if err != nil {
					continue
				}

				if accrualResp.Status == "PROCESSED" && accrualResp.Accrual > 0 {
					err = storage.UpdateBalance(order.UserID, accrualResp.Accrual)
					if err != nil {
					}
				}
			}
		}
	}
}
