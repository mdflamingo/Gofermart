package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/mdflamingo/Gofermart/internal/config"
	"github.com/mdflamingo/Gofermart/internal/handler"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	conf := config.ParseFlags()
	if err := run(conf); err != nil {
		log.Fatal(err)
	}
}

func run(conf *config.Config) error {
	if err := logger.Initialize(conf.LogLevel); err != nil {
		return err
	}

	if conf.CookieSecretKey == "" {
		logger.Log.Fatal("CookieSecretKey is required")
	}

	logger.Log.Info("Running server", zap.String("address", conf.RunAddr))

	storage, err := initStorage(conf)
	if err != nil {
		logger.Log.Fatal("Failed to create storage", zap.Error(err))
	}
	defer storage.Close()

	worker := handler.NewAccrualWorker(conf.AccrualHost, storage)
	go worker.Start(context.Background())

	r := handler.NewRouter(conf, storage, worker)

	return http.ListenAndServe(conf.RunAddr, r)
}

func initStorage(conf *config.Config) (*repository.DBStorage, error) {
	if conf.DataBaseDSN == "" {
		return nil, errors.New("DATABASE_DSN is required")
	}

	logger.Log.Info("Attempting to use database storage", zap.String("dsn", conf.DataBaseDSN))
	storage, err := repository.NewDBStorage(conf.DataBaseDSN)
	if err != nil {
		logger.Log.Warn("Failed to initialize database storage", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Successfully initialized database storage")
	return storage, nil
}
