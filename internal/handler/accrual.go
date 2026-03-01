package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"
)

type AccrualWorker struct {
	client  *resty.Client
	storage *repository.DBStorage
}

func NewAccrualWorker(accrualURL string, storage *repository.DBStorage) *AccrualWorker {
	return &AccrualWorker{
		client:  resty.New().SetBaseURL(accrualURL),
		storage: storage,
	}
}

func (w *AccrualWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := w.storage.GetOrdersToUpdate()
			if err != nil {
				zap.L().Error("Failed to get orders to update", zap.Error(err))
				continue
			}

			for _, order := range orders {
				resp, err := w.client.R().Get("/api/orders/" + order.Number)
				if err != nil || resp.StatusCode() != 200 {
					zap.L().Warn("Failed to get order from accrual", zap.String("order", order.Number), zap.Error(err))
					continue
				}

				var accrualResp struct {
					Status  string  `json:"status"`
					Accrual float64 `json:"accrual"`
				}
				json.Unmarshal(resp.Body(), &accrualResp)

				err = w.storage.UpdateOrderStatus(order.ID, accrualResp.Status, accrualResp.Accrual)
				if err != nil {
					zap.L().Error("Failed to update order status", zap.Error(err))
					continue
				}

				if accrualResp.Status == "PROCESSED" && accrualResp.Accrual > 0 {
					zap.L().Info("Updating balance", zap.Float64("accrual", accrualResp.Accrual), zap.Int("userID", order.UserID))
					err = w.storage.UpdateBalance(order.UserID, accrualResp.Accrual)
					if err != nil {
						zap.L().Error("Failed to update balance", zap.Error(err))
					}
				}
			}
		}
	}
}
