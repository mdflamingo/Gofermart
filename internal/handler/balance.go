package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/middleware"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"
)

func GetBalanceHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	userID, err := middleware.GetUserIDFromRequest(request)
	if err != nil {
		logger.Log.Warn("failed to get userID", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	balanceDB, err := storage.GetBalance(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Log.Error("The balance does not exist for the incoming user", zap.Error(err))
			http.Error(response, "User not found", http.StatusNotFound)
			return
		}
		logger.Log.Error("Failed to get balance", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if balanceDB == (repository.Balance{}) {
		http.Error(response, "User not found", http.StatusNotFound)
		return
	}

	balanceResponse := models.BalanceResponse{
		Current:   balanceDB.Current,
		Withdrawn: balanceDB.Withdrawn,
	}

	respJSON, err := json.Marshal(balanceResponse)
	if err != nil {
		logger.Log.Error("Failed to marshal response to JSON", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	response.Write(respJSON)
}

func WithdrawalsHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	if request.Header.Get("Content-Type") != "application/json" {
		logger.Log.Warn("invalid content type", zap.String("content_type", request.Header.Get("Content-Type")))
		http.Error(response, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	userID, err := middleware.GetUserIDFromRequest(request)
	if err != nil {
		logger.Log.Warn("failed to get userID", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var withdraw models.WithdrawnRequest
	var buf bytes.Buffer

	_, err = buf.ReadFrom(request.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &withdraw); err != nil {
		logger.Log.Error("Failed to unmarshal JSON",
			zap.Error(err),
			zap.String("request_body", buf.String()))
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	orderNum := strings.TrimSpace(withdraw.Order)
	if !checkOrderNum(orderNum) {
		logger.Log.Error("Incorrect order number format")
		http.Error(response, "Incorrect order number format", http.StatusUnprocessableEntity)
		return
	}

	balanceDB, err := storage.GetBalance(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Log.Error("The balance does not exist for the incoming user", zap.Error(err))
			http.Error(response, "User not found", http.StatusNotFound)
			return
		}
		logger.Log.Error("Failed to get balance", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if balanceDB == (repository.Balance{}) {
		http.Error(response, "User not found", http.StatusNotFound)
		return
	}

	if float64(withdraw.Sum) > balanceDB.Current {
		logger.Log.Error("there are insufficient funds in the account")
		http.Error(response, "Insufficient funds", http.StatusPaymentRequired)
		return
	}

	err = storage.SaveWithdrawal(userID, orderNum, float64(withdraw.Sum))
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			http.Error(response, "Insufficient funds", http.StatusPaymentRequired)
			return
		}
		logger.Log.Error("Failed to save withdrawal", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "text/plain")
	response.WriteHeader(http.StatusOK)
}

func GetWithdrawalsHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	userID, err := middleware.GetUserIDFromRequest(request)
	if err != nil {
		logger.Log.Warn("failed to get userID", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	withdrawalsDB, err := storage.GetWithdrawals(userID)
	if err != nil {
		logger.Log.Error("Failed to get withdrawals", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(withdrawalsDB) == 0 {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusNoContent)
		return
	}

	responses := make([]models.WithdrawnResponse, 0, len(withdrawalsDB))

	for _, obj := range withdrawalsDB {
		responses = append(responses, models.WithdrawnResponse{
			Order:       obj.Order,
			Sum:         obj.Sum,
			ProcessedAt: obj.ProcessedAt,
		})
	}

	respJSON, err := json.Marshal(responses)
	if err != nil {
		logger.Log.Error("Failed to marshal response to JSON", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	response.Write(respJSON)
}
