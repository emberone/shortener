package main

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shortener/internal/config"
	"shortener/internal/handler"
	"shortener/internal/service"
)

func main() {

	config.Configure()

	if a := os.Getenv("SERVER_ADDRESS"); a != "" {
		config.Server.A = a
	}
	if b := os.Getenv("BASE_URL"); b != "" {
		config.Server.B = b
	}
	if f := os.Getenv("FILE_STORAGE_PATH"); f != "" {
		config.Server.Storage = f
	}

	service.ReadFromFile()

	p, _ := url.Parse(config.Server.B)
	config.Server.Path = p.Path

	if config.Server.Path == "" {
		config.Server.Path = "/"
	}
	config.Server.Host = p.Host
	config.Server.Scheme = p.Scheme

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
	})

	r.Post("/api/shorten", handler.PutLinkAPIHandler)

	log.Fatal(http.ListenAndServe(config.Server.A, r))
}
