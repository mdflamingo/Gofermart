package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	accrualURL := fmt.Sprintf("http://%s:%s", conf.AccrualHost, conf.AccrualPort)
	logger.Log.Info("Accrual URL", zap.String("url", accrualURL))

	handler.InitAccrualClient(accrualURL)
	handler.InitAccrualClient(accrualURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handler.StartAccrualWorker(ctx, storage)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Log.Info("Shutting down server")
		cancel()
	}()

	r := handler.NewRouter(conf, storage)

	server := &http.Server{
		Addr:    conf.RunAddr,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server shutdown failed", zap.Error(err))
		return err
	}

	logger.Log.Info("Server stopped")
	return nil
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
