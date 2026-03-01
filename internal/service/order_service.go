package service

import (
	"errors"

	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
)

var (
	ErrInvalidOrderNumber         = errors.New("invalid order number")
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by this user")
	ErrOrderAlreadyUploadedByOther = errors.New("order already uploaded by another user")
	ErrOrderNotFound              = errors.New("order not found")
)

type OrderService struct {
	repo *repository.DBStorage
}

func NewOrderService(repo *repository.DBStorage) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) UploadOrder(orderNum string, userID int) (int, error) {
	if err := ValidateOrderNumber(orderNum); err != nil {
		return 0, err
	}

	ownerID, err := s.repo.SaveOrder(orderNum, userID)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			if ownerID == userID {
				return ownerID, ErrOrderAlreadyUploadedByUser
			}
			return ownerID, ErrOrderAlreadyUploadedByOther
		}
		return 0, err
	}

	return ownerID, nil
}

func (s *OrderService) GetUserOrders(userID int) ([]models.OrdersResponse, error) {
	orders, err := s.repo.GetOrders(userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.OrdersResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, models.OrdersResponse{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		})
	}

	return responses, nil
}

func (s *OrderService) GetOrder(orderNum string) (*models.OrderInfoResponse, error) {
	order, err := s.repo.GetOrder(orderNum)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	resp := &models.OrderInfoResponse{
		Number: order.Number,
		Status: order.Status,
	}

	if order.Accrual != 0 {
		resp.Accrual = &order.Accrual
	}

	return resp, nil
}
