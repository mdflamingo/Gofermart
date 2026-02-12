package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"
)

var ErrEmptyRequiredField = errors.New("login and password cannot be empty")
var ErrJSONFormat = errors.New("invalid JSON format")
var ErrRequestRead = errors.New("failed to read request body")

func AuthorizationHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	userDB, err := parseBody(response, request)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	err = storage.SaveUser(userDB)
	if err != nil {
		logger.Log.Error("failed to save user", zap.Error(err))

		if errors.Is(err, repository.ErrConflict) {
			http.Error(response, "User already exists", http.StatusConflict)
			return
		}

		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
}
func AuthenticationHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage, secretKey string) {
	userDB, err := parseBody(response, request)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := storage.GetUser(userDB)
	if err != nil {
		logger.Log.Error("failed to save user", zap.Error(err))

		if errors.Is(err, repository.ErrNotFound) {
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	jwtToken, err := createJWT(userID, secretKey)
	if err != nil {
		logger.Log.Error("failed to create JWT token", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(response, &http.Cookie{
		Name:     "token",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
}

func createJWT(userID int, secretKey string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		logger.Log.Error("Error creating JWT", zap.Error(err))
		return "", err
	}
	return tokenString, nil
}

func parseBody(_ http.ResponseWriter, request *http.Request) (models.UserDB, error) {
	var requestUser models.AuthUser
	var buf bytes.Buffer

	_, err := buf.ReadFrom(request.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		return models.UserDB{}, ErrRequestRead
	}

	if err = json.Unmarshal(buf.Bytes(), &requestUser); err != nil {
		logger.Log.Error("Failed to unmarshal JSON",
			zap.Error(err),
			zap.String("request_body", buf.String()))
		return models.UserDB{}, ErrJSONFormat
	}

	if requestUser.Login == "" || requestUser.Password == "" {
		return models.UserDB{}, ErrEmptyRequiredField
	}

	h := sha256.New()
	h.Write([]byte(requestUser.Password))
	hashedPassword := h.Sum(nil)

	userDB := models.UserDB{
		Login:    requestUser.Login,
		Password: hex.EncodeToString(hashedPassword),
	}
	return userDB, nil
}
