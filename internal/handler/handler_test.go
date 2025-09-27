package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/kuznet1/gophermart/internal/config"
	"github.com/kuznet1/gophermart/internal/middleware"
	"github.com/kuznet1/gophermart/internal/model"
	"github.com/kuznet1/gophermart/internal/repository"
	"github.com/kuznet1/gophermart/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/theplant/luhn"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var userName = generateUserName()
var orderID = generateOrderID()

func generateUserName() string {
	return fmt.Sprintf("user%d", time.Now().Unix())
}

func generateOrderID() int {
	base := int(time.Now().Unix())
	check := luhn.CalculateLuhn(base)
	return base*10 + check
}

func TestFlow(t *testing.T) {
	mux, err := newMux()
	require.NoError(t, err)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := &http.Client{}
	var cookies []*http.Cookie

	t.Run("register", func(t *testing.T) {
		cred := model.UserCredentials{Login: userName, Password: "pass1"}
		b, _ := json.Marshal(cred)
		resp, err := http.Post(ts.URL+"/api/user/register", "application/json", bytes.NewBuffer(b))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		cookies = resp.Cookies()
	})

	t.Run("login", func(t *testing.T) {
		cred := model.UserCredentials{Login: userName, Password: "pass1"}
		b, _ := json.Marshal(cred)
		resp, err := http.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(b))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		cookies = resp.Cookies()
	})

	t.Run("new order", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/user/orders", bytes.NewBufferString(strconv.Itoa(orderID)))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("get orders", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/user/orders", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()
		var orders []model.Order
		err = json.NewDecoder(resp.Body).Decode(&orders)
		require.NoError(t, err)
		require.NotEmpty(t, orders)
		require.Equal(t, orderID, orders[0].Order)
	})

	t.Run("get balance", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/user/balance", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()
		var bal model.Balance
		err = json.NewDecoder(resp.Body).Decode(&bal)
		require.NoError(t, err)
		require.Equal(t, bal.Current, float64(1))
	})

	t.Run("withdraw", func(t *testing.T) {
		withdrawReq := model.Withdraw{
			Order: orderID,
			Sum:   1,
		}
		b, _ := json.Marshal(withdrawReq)
		req, _ := http.NewRequest("POST", ts.URL+"/api/user/balance/withdraw", bytes.NewBuffer(b))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("get withdrawals", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/user/withdrawals", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Logf("withdrawals: %s", body)

		var withdrawals []model.Withdrawal
		err = json.Unmarshal(body, &withdrawals)
		require.NoError(t, err)
		require.NotEmpty(t, withdrawals)
		require.Equal(t, orderID, withdrawals[0].Order)
	})
}

func newMux() (*chi.Mux, error) {
	cfg := config.Config{
		RunAddress:           ":8086",
		AccrualSystemAddress: "http://localhost:8080",
		DatabaseURI:          "postgres://postgres@localhost:5432/gophermart",
		MigrationsPath:       "file://../../migrations",
	}

	db, err := repository.InitDBConnection(cfg)
	if err != nil {
		return nil, err
	}

	repo := repository.NewRepo(db)
	accrualClient := &accrualMock{repo: repo}
	auth := middleware.NewAuth(cfg)
	svc := service.NewService(repo, auth, accrualClient)
	s := NewHandler(svc, auth)
	return s.Router(), nil
}

type accrualMock struct {
	repo *repository.Repo
}

func (p *accrualMock) Signal() {
	orders, _ := p.repo.GetProcessingOrders()
	for _, order := range orders {
		p.repo.UpdateAccrual(model.AccrualResp{Order: order, Status: "PROCESSED", Accrual: 1})
	}
}
