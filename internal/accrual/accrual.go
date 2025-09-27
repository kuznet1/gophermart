package accrual

import (
	"encoding/json"
	"fmt"
	"github.com/kuznet1/gophermart/internal/model"
	"github.com/kuznet1/gophermart/internal/repository"
	"io"
	"net/http"
)

type Accrualer interface {
	Signal()
}

type Accrual struct {
	signal               chan struct{}
	accrualSystemAddress string
	repo                 *repository.Repo
}

func NewAccrual(accrualSystemAddress string, repo *repository.Repo) *Accrual {
	return &Accrual{
		signal:               make(chan struct{}, 1),
		accrualSystemAddress: accrualSystemAddress,
		repo:                 repo,
	}
}

func (a *Accrual) Start() {
	go a.run()
	a.Signal()
}

func (a *Accrual) Stop() {
	close(a.signal)
}

func (a *Accrual) run() {
	for range a.signal {
		orders, err := a.repo.GetProcessingOrders()
		if err != nil {
			a.Signal()
			continue
		}
		for _, order := range orders {
			if err = a.updateOrderData(order); err != nil {
				a.Signal()
			}
		}
	}
}

func (a *Accrual) Signal() {
	select {
	case a.signal <- struct{}{}:
	default:
	}
}

func (a *Accrual) updateOrderData(order int) error {
	url := fmt.Sprintf("%s/api/orders/%d", a.accrualSystemAddress, order)
	response, err := http.Get(url)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		payload, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		var accrual model.AccrualResp
		if err = json.Unmarshal(payload, &accrual); err != nil {
			return err
		}

		err = a.repo.UpdateAccrual(accrual)
		if err != nil {
			return err
		}
	case http.StatusTooManyRequests:
		a.Signal()
	case http.StatusInternalServerError:
		a.Signal()
	}

	return nil
}
