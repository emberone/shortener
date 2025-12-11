package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shortener/internal/config"
	"shortener/internal/config/db"
	"shortener/internal/handler"
	"shortener/internal/service"
)

func main() {

	config.Configure()

	db.Open("sqlite", "videos.db")

	db.Ping()

	// row := db.DB.QueryRow("select version();")

	// var res string
	// row.Scan(&res)

	os.Exit(0)

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
