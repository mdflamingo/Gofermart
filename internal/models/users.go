package models

type AuthUser struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

type UserDB struct {
	Login string
	Password string
}

