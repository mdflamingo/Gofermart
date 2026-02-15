package models

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64     `json:"withdrawn"`
}

type WithdrawnRequest struct {
	Order string `json:"order"`
	Sum   int    `json:"sum"`
}
