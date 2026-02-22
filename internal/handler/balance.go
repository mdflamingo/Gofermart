package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/middleware"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/service"
	"go.uber.org/zap"
)

func GetBalanceHandler(w http.ResponseWriter, r *http.Request, svc *service.BalanceService) {
	userID, err := middleware.GetUserIDFromRequest(r)
	if err != nil {
		logger.Log.Warn("failed to get user ID", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	balance, err := svc.GetBalance(userID)
	if err != nil {
		if errors.Is(err, service.ErrBalanceNotFound) {
			logger.Log.Warn("balance not found for user", zap.Int("user_id", userID))
			http.Error(w, "Balance not found", http.StatusNotFound)
			return
		}
		logger.Log.Error("failed to get balance", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(balance)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func WithdrawHandler(w http.ResponseWriter, r *http.Request, svc *service.BalanceService) {
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Warn("invalid content type", zap.String("content_type", r.Header.Get("Content-Type")))
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	userID, err := middleware.GetUserIDFromRequest(r)
	if err != nil {
		logger.Log.Warn("failed to get user ID", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var req models.WithdrawnRequest
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Log.Warn("failed to unmarshal request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Sum <= 0 {
		logger.Log.Warn("invalid withdrawal sum", zap.Float64("sum", req.Sum))
		http.Error(w, "Sum must be positive", http.StatusBadRequest)
		return
	}

	err = svc.Withdraw(userID, req.Order, req.Sum)
	if err != nil {
		handleWithdrawError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleWithdrawError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidOrderNumber):
		logger.Log.Warn("incorrect order number format")
		http.Error(w, "Incorrect order number format", http.StatusUnprocessableEntity)
	case errors.Is(err, service.ErrBalanceNotFound):
		logger.Log.Warn("balance not found")
		http.Error(w, "Balance not found", http.StatusNotFound)
	case errors.Is(err, service.ErrInsufficientFunds):
		logger.Log.Warn("insufficient funds for withdrawal")
		http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
	default:
		logger.Log.Error("failed to process withdrawal", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func GetWithdrawalsHandler(w http.ResponseWriter, r *http.Request, svc *service.BalanceService) {
	userID, err := middleware.GetUserIDFromRequest(r)
	if err != nil {
		logger.Log.Warn("failed to get user ID", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	withdrawals, err := svc.GetWithdrawals(userID)
	if err != nil {
		logger.Log.Error("failed to get withdrawals", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if len(withdrawals) == 0 {
		w.Write([]byte("[]"))
		return
	}

	respJSON, err := json.Marshal(withdrawals)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Write(respJSON)
}
