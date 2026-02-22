package service

import (
	"errors"

	"github.com/mdflamingo/Gofermart/internal/models"
	"github.com/mdflamingo/Gofermart/internal/repository"
)

var (
	ErrBalanceNotFound   = errors.New("balance not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type BalanceService struct {
	repo *repository.DBStorage
}

func NewBalanceService(repo *repository.DBStorage) *BalanceService {
	return &BalanceService{repo: repo}
}

func (s *BalanceService) GetBalance(userID int) (*models.BalanceResponse, error) {
	balance, err := s.repo.GetBalance(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBalanceNotFound
		}
		return nil, err
	}

	return &models.BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}, nil
}

func (s *BalanceService) Withdraw(userID int, orderNum string, sum float64) error {
	if err := ValidateOrderNumber(orderNum); err != nil {
		return ErrInvalidOrderNumber
	}

	balance, err := s.repo.GetBalance(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrBalanceNotFound
		}
		return err
	}

	if sum > balance.Current {
		return ErrInsufficientFunds
	}

	err = s.repo.SaveWithdrawal(userID, orderNum, sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return err
	}

	return nil
}

func (s *BalanceService) GetWithdrawals(userID int) ([]models.WithdrawnResponse, error) {
	withdrawals, err := s.repo.GetWithdrawals(userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.WithdrawnResponse, 0, len(withdrawals))
	for _, w := range withdrawals {
		responses = append(responses, models.WithdrawnResponse{
			Order:       w.Order,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt,
		})
	}

	return responses, nil
}
