package service

import (
	"context"
	"github.com/kuznet1/gophermart/internal/accrual"
	"github.com/kuznet1/gophermart/internal/errs"
	"github.com/kuznet1/gophermart/internal/middleware"
	"github.com/kuznet1/gophermart/internal/model"
	"github.com/kuznet1/gophermart/internal/repository"
	"github.com/theplant/luhn"
	"strconv"
)

type Service struct {
	repo    *repository.Repo
	auth    *middleware.Auth
	accrual accrual.Accrualer
}

func NewService(repo *repository.Repo, auth *middleware.Auth, accrual accrual.Accrualer) *Service {
	return &Service{repo, auth, accrual}
}

func (s *Service) NewOrder(ctx context.Context, order string) error {
	userID, err := s.auth.GetUserID(ctx)
	if err != nil {
		return err
	}

	orderID, err := strconv.Atoi(order)
	if err != nil {
		return errs.ErrInvalidOrderNum
	}

	if !luhn.Valid(orderID) {
		return errs.ErrInvalidOrderNum
	}

	err = s.repo.AddOrder(userID, orderID)
	s.accrual.Signal()
	return err
}

func (s *Service) GetOrders(ctx context.Context) ([]model.Order, error) {
	userID, err := s.auth.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.GetOrders(userID)
}

func (s *Service) GetBalance(ctx context.Context) (model.Balance, error) {
	userID, err := s.auth.GetUserID(ctx)
	if err != nil {
		return model.Balance{}, err
	}

	return s.repo.GetBalance(userID)
}

func (s *Service) Withdraw(ctx context.Context, withdraw model.Withdraw) error {
	userID, err := s.auth.GetUserID(ctx)
	if err != nil {
		return err
	}

	if !luhn.Valid(withdraw.Order) {
		return errs.ErrInvalidOrderNum
	}

	return s.repo.NewWithdrawal(userID, withdraw)
}

func (s *Service) GetWithdrawals(ctx context.Context) ([]model.Withdrawal, error) {
	userID, err := s.auth.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.GetWithdrawals(userID)
}

func (s *Service) Login(creds model.UserCredentials) (string, error) {
	userID, err := s.repo.Login(creds)
	if err != nil {
		return "", err
	}

	return s.auth.CreateToken(userID)
}

func (s *Service) Register(creds model.UserCredentials) (string, error) {
	userID, err := s.repo.Register(creds)
	if err != nil {
		return "", err
	}

	return s.auth.CreateToken(userID)
}
