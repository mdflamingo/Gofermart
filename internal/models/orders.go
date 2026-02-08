package models

import "time"

type OrdersResponse struct {
	Number string `json:"number"`
	Status string `json:"status"`
	Accrual int `json:"accrual"`
	Uploaded_at time.Time `json:"uploaded_at"`
}
