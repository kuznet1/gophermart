package handler

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/kuznet1/gophermart/internal/errs"
	"github.com/kuznet1/gophermart/internal/logger"
	"github.com/kuznet1/gophermart/internal/middleware"
	"github.com/kuznet1/gophermart/internal/model"
	"github.com/kuznet1/gophermart/internal/service"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Handler struct {
	svc  *service.Service
	auth *middleware.Auth
}

func NewHandler(svc *service.Service, auth *middleware.Auth) *Handler {
	return &Handler{svc, auth}
}

func (h *Handler) Router() *chi.Mux {
	r := chi.NewMux()
	r.Use(middleware.Compression)

	r.Route("/api", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)

			r.Group(func(r chi.Router) {
				r.Use(h.auth.Authentication)
				r.Post("/orders", h.NewOrder)
				r.Get("/orders", h.GetOrders)
				r.Get("/balance", h.GetBalance)
				r.Post("/balance/withdraw", h.Withdraw)
				r.Get("/withdrawals", h.GetWithdrawals)
			})

		})
	})
	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var user model.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.svc.Register(user)
	var httpErr *errs.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.Code())
		return
	}

	if err != nil {
		internalError(err, w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  middleware.AuthCookieName,
		Value: token,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var user model.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(user)
	var httpErr *errs.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.Code())
		return
	}

	if err != nil {
		internalError(err, w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  middleware.AuthCookieName,
		Value: token,
	})
}

func (h *Handler) NewOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order := string(body)

	err = h.svc.NewOrder(r.Context(), order)
	var httpErr *errs.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.Code())
		return
	}

	if err != nil {
		internalError(err, w)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.GetOrders(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	if len(orders) == 0 {
		status = http.StatusNoContent
	}

	respJSON(w, orders, status)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	balance, err := h.svc.GetBalance(r.Context())
	if err != nil {
		internalError(err, w)
		return
	}

	respJSON(w, balance, http.StatusOK)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var withdraw model.Withdraw
	if err := json.NewDecoder(r.Body).Decode(&withdraw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.svc.Withdraw(r.Context(), withdraw)
	var httpErr *errs.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.Code())
		return
	}

	if err != nil {
		internalError(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	withdrawals, err := h.svc.GetWithdrawals(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	if len(withdrawals) == 0 {
		status = http.StatusNoContent
	}

	respJSON(w, withdrawals, status)
}

func internalError(err error, w http.ResponseWriter) {
	logger.Log.Error(err.Error(), zap.Error(err))
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func respJSON(w http.ResponseWriter, resp any, code int) {
	data, err := json.Marshal(resp)
	if err != nil {
		internalError(err, w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
