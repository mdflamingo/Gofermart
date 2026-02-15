package models

import "time"

type WithdrawnResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type WithdrawnRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
