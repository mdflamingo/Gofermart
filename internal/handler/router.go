package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mdflamingo/Gofermart/internal/config"
	"github.com/mdflamingo/Gofermart/internal/logger"
	"github.com/mdflamingo/Gofermart/internal/repository"
)

func NewRouter(conf *config.Config, storage *repository.DBStorage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger)

	// r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
	// 	handler.DBHealthCheck(w, r, storage)
	// })
	// r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.GetHandler(w, req, storage)
	// })
	// r.Post("/", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.PostHandler(w, req, conf.BaseShortURL, storage)
	// })
	// r.Post("/api/shorten", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.JSONPostHandler(w, req, conf.BaseShortURL, storage)
	// })
	// r.Post("/api/shorten/batch", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.BatchHandler(w, req, conf.BaseShortURL, storage)
	// })
	// r.Get("/api/user/urls", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.GetUserURLSHandler(w, req, conf.BaseShortURL, storage)
	// })
	// r.Delete("/api/user/urls", func(w http.ResponseWriter, req *http.Request) {
	// 	handler.DeleteUserURLSHandler(w, req, conf.BaseShortURL, storage)
	// })

	return r
}
