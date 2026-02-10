package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/middleware"
	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"
)

func UploadOrderNumHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	if request.Header.Get("Content-Type") != "text/plain" {
		logger.Log.Warn("invalid content type", zap.String("content_type", request.Header.Get("Content-Type")))
		http.Error(
			response,
			"Invalid Content-Type",
			http.StatusUnsupportedMediaType,
		)
		return
	}

	userID, err := middleware.GetUserIDFromRequest(request)
	if err != nil {
		logger.Log.Warn("failed to get userID", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(request.Body)

	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	orderNum := strings.TrimSpace(string(body))
	if !checkOrderNum(orderNum) {
		logger.Log.Error("Incorrect order number format", zap.Error(err))
		http.Error(response, "Incorrect order number format", http.StatusUnprocessableEntity)
		return
	}

	userDB, err := storage.SaveOrder(orderNum, userID)

	if err != nil {
		if errors.Is(err, repository.ErrConflict) && userDB == userID {
			logger.Log.Error("The order number has already been uploaded by this user", zap.Error(err))
			http.Error(response, "The order number has already been uploaded by this user", http.StatusOK)
			return
		}
		if errors.Is(err, repository.ErrConflict) && userDB != userID {
			logger.Log.Error("The order number has already been uploaded by another user", zap.Error(err))
			http.Error(response, "The order number has already been uploaded by another user", http.StatusConflict)
			return
		}
		logger.Log.Error("msg=", zap.Error(err))
        http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

    response.Header().Set("Content-Type", "text/plain")
    response.WriteHeader(http.StatusAccepted)
}


func checkOrderNum(number string) bool {
	sum := 0
	isSecond := false

	for i := len(number) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false
		}

		if isSecond {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}

func GetOrdersHandler(response http.ResponseWriter, request *http.Request, storage *repository.DBStorage) {
	userID, err := middleware.GetUserIDFromRequest(request)
	if err != nil {
		logger.Log.Warn("failed to get userID", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	orders, err := storage.GetOrders(userID)
	if err != nil {
		logger.Log.Error("Failed to get orders", zap.Error(err))
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusNoContent)
		return
	}

	responses := make([]models.OrdersResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, models.OrdersResponse {
			Number: order.Number,
			Status: order.Status,
			Accrual: order.Accrual,
			Uploaded_at: order.Uploaded_at,
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
