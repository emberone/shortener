package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shortener/internal/config"
	"shortener/internal/handler"
	"shortener/internal/service"
)

func main() {

	config.Configure()

	service.ReadFromFile()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	config.Sugar = *logger.Sugar()

	r := chi.NewRouter()
	r.Use(handler.WithLogging, handler.GzipHandle)

	r.Route(config.Server.Path, func(r chi.Router) {
		r.Get("/{linkid}", handler.GetLinkHandler)
		r.Post("/", handler.PutLinkHandler)
		r.Get("/ping", handler.DBHandler)
	})

	r.Post("/api/shorten", handler.PutLinkAPIHandler)

	log.Fatal(http.ListenAndServe(config.Server.A, r))
}
