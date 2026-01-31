package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"
)

func AuthorizationHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
    var requestUser models.AuthorizationUser
    var buf bytes.Buffer

    _, err := buf.ReadFrom(request.Body)
    if err != nil {
        logger.Log.Error("failed to read request body", zap.Error(err))
        http.Error(response, err.Error(), http.StatusBadRequest)
        return
    }

    if err = json.Unmarshal(buf.Bytes(), &requestUser); err != nil {
        logger.Log.Error("Failed to unmarshal JSON",
            zap.Error(err),
            zap.String("request_body", buf.String()))
        http.Error(response, err.Error(), http.StatusBadRequest)
        return
    }

    if requestUser.Login == "" || requestUser.Password == "" {
        http.Error(response, "Login and Password cannot be empty", http.StatusBadRequest)
        return
    }

    h := sha256.New()
    h.Write([]byte(requestUser.Password))
    hashedPassword := h.Sum(nil)

    userDB := models.UserDB{
        Login:    requestUser.Login,
        Password: hex.EncodeToString(hashedPassword),
    }

    err = storage.Save(userDB)
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
