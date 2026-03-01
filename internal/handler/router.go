package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mdflamingo/Gofermart/internal/config"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/middleware"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"github.com/mdflamingo/Gofermart/internal/service"
)

func NewRouter(conf *config.Config, storage *repository.DBStorage, worker *AccrualWorker) *chi.Mux {
	r := chi.NewRouter()

	orderService := service.NewOrderService(storage)
	balanceService := service.NewBalanceService(storage)
	userService := service.NewUserService(storage)

	r.Use(logger.RequestLogger)

	r.Group(func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			DBHealthCheck(w, r, storage)
		})

		r.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
			AuthorizationHandler(w, r, userService, conf.CookieSecretKey)
		})

		r.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
			AuthenticationHandler(w, r, userService, conf.CookieSecretKey)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(conf.CookieSecretKey))

		r.Post("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
			UploadOrderHandler(w, r, orderService)
		})
		r.Get("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
			GetOrdersHandler(w, r, orderService)
		})
		r.Get("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
			GetBalanceHandler(w, r, balanceService)
		})
		r.Post("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
			WithdrawHandler(w, r, balanceService)
		})
		r.Get("/api/user/withdrawals", func(w http.ResponseWriter, r *http.Request) {
			GetWithdrawalsHandler(w, r, balanceService)
		})
		r.Get("/api/user/{number}", rateLimit(func(w http.ResponseWriter, r *http.Request) {
			GetOrderHandler(w, r, orderService)
		}))
	})

	return r
}
