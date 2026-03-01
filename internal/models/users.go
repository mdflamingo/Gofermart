package models

type AuthUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserDB struct {
	Login    string
	Password string
}

type User struct {
	ID    int
	Login string
	Token string
}

type AuthResponse struct {
	Token string `json:"token"`
}
