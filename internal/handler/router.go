package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mdflamingo/Gofermart/internal/config"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/repository"
	"github.com/mdflamingo/Gofermart/internal/middleware"
)

func NewRouter(conf *config.Config, storage *repository.DBStorage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger)

	r.Group(func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			DBHealthCheck(w, r, storage)
		})

		r.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
			AuthorizationHandler(w, r, storage)
		})

		r.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
			AuthenticationHandler(w, r, storage, conf.CookieSecretKey)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(conf.CookieSecretKey))

		r.Post("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
			UploadOrderNumHandler(w, r, storage)
		})
		r.Get("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
			GetOrders(w, r, storage)
		})
	})

	return r
}
