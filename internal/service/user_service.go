package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
)

var (
	ErrEmptyRequiredField = errors.New("login and password cannot be empty")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserService struct {
	repo *repository.DBStorage
}

func NewUserService(repo *repository.DBStorage) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(user models.AuthUser) (int, error) {
	if user.Login == "" || user.Password == "" {
		return 0, ErrEmptyRequiredField
	}

	hashedPassword := hashPassword(user.Password)

	userDB := models.UserDB{
		Login:    user.Login,
		Password: hashedPassword,
	}

	userID, err := s.repo.SaveUser(userDB)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return 0, ErrUserAlreadyExists
		}
		return 0, err
	}

	if err := s.repo.InitBalance(userID); err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *UserService) Login(user models.AuthUser) (int, error) {
	if user.Login == "" || user.Password == "" {
		return 0, ErrEmptyRequiredField
	}

	hashedPassword := hashPassword(user.Password)

	userDB := models.UserDB{
		Login:    user.Login,
		Password: hashedPassword,
	}

	userID, err := s.repo.GetUser(userDB)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	return userID, nil
}

func (s *UserService) GenerateToken(userID int, secretKey string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func hashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
