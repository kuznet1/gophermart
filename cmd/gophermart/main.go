package main

import (
	"database/sql"
	"github.com/kuznet1/gophermart/internal/accrual"
	"github.com/kuznet1/gophermart/internal/config"
	"github.com/kuznet1/gophermart/internal/handler"
	"github.com/kuznet1/gophermart/internal/logger"
	"github.com/kuznet1/gophermart/internal/middleware"
	"github.com/kuznet1/gophermart/internal/repository"
	"github.com/kuznet1/gophermart/internal/service"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Log.Fatal("unable to parse config", zap.Error(err))
	}

	db, err := repository.InitDBConnection(cfg)
	if err != nil {
		logger.Log.Fatal("failed to init sql connection", zap.Error(err))
	}

	startService(db, cfg)
}

func startService(db *sql.DB, cfg config.Config) {
	repo := repository.NewRepo(db)
	acc := accrual.NewAccrual(cfg.AccrualSystemAddress, repo)
	acc.Start()
	defer acc.Stop()
	auth := middleware.NewAuth(cfg)
	svc := service.NewService(repo, auth, acc)
	h := handler.NewHandler(svc, auth)

	logger.Log.Info("Gophermart service is running at " + cfg.RunAddress)
	logger.Log.Fatal(http.ListenAndServe(cfg.RunAddress, h.Router()).Error())
}
