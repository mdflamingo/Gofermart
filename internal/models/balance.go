package models


type BalanceResponse struct {
	Current float64 `json:"current"`
	Withdrawn int `json:"withdrawn"`
}

type WithdrawnRequest struct {
	Order string `json:"order"`
	Sum int `json:"sum"`
}
