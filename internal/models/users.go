package models

type AuthorizationUser struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

type UserDB struct {
	Login string
	Password string
}

type Response struct {
	Result string `json:"result"`
}

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ResponseByUser struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}
