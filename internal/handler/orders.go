package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/middleware"
	"github.com/mdflamingo/Gofermart/internal/service"
	"go.uber.org/zap"
)

func UploadOrderHandler(w http.ResponseWriter, r *http.Request, svc *service.OrderService) {
	if r.Header.Get("Content-Type") != "text/plain" {
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

	orderNum := strings.TrimSpace(string(body))

	_, err = svc.UploadOrder(orderNum, userID)
	if err != nil {
		handleOrderUploadError(w, orderNum, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusAccepted)
}

func handleOrderUploadError(w http.ResponseWriter, orderNum string, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidOrderNumber):
		logger.Log.Warn("incorrect order number format", zap.String("order", orderNum))
		http.Error(w, "Incorrect order number format", http.StatusUnprocessableEntity)
	case errors.Is(err, service.ErrOrderAlreadyUploadedByUser):
		logger.Log.Info("order already uploaded by user", zap.String("order", orderNum))
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrOrderAlreadyUploadedByOther):
		logger.Log.Warn("order already uploaded by another user", zap.String("order", orderNum))
		http.Error(w, "The order number has already been uploaded by another user", http.StatusConflict)
	default:
		logger.Log.Error("failed to save order", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func GetOrdersHandler(w http.ResponseWriter, r *http.Request, svc *service.OrderService) {
	userID, err := middleware.GetUserIDFromRequest(r)
	if err != nil {
		logger.Log.Warn("failed to get user ID", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orders, err := svc.GetUserOrders(userID)
	if err != nil {
		logger.Log.Error("failed to get orders", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respJSON, err := json.Marshal(orders)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func GetOrderHandler(w http.ResponseWriter, r *http.Request, svc *service.OrderService) {
	_, err := middleware.GetUserIDFromRequest(r)
	if err != nil {
		logger.Log.Warn("failed to get user ID", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderNum := chi.URLParam(r, "number")
	if orderNum == "" {
		logger.Log.Warn("order number is empty")
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	order, err := svc.GetOrder(orderNum)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			logger.Log.Warn("order not found", zap.String("order", orderNum))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		logger.Log.Error("failed to get order", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(order)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}
