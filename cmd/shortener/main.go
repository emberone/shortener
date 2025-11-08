package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var storage map[string]string = map[string]string{}

var server struct {
	a      string
	b      string
	path   string
	host   string
	scheme string
}

var sugar zap.SugaredLogger

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

type raw struct {
	URL string `json:"url"`
}

func main() {

	flag.StringVar(&server.a, "a", "localhost:8080", "server address and port")
	flag.StringVar(&server.b, "b", "http://localhost:8080/", "server address and port")
	flag.Parse()

	if a := os.Getenv("SERVER_ADDRESS"); a != "" {
		server.a = a
	}
	if b := os.Getenv("SERVER_bDDRESS"); b != "" {
		server.b = b
	}

	p, _ := url.Parse(server.b)
	server.path = p.Path

	if server.path == "" {
		server.path = "/"
	}
	server.host = p.Host
	server.scheme = p.Scheme

	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer logger.Sync()

	sugar = *logger.Sugar()

	r := chi.NewRouter()
	r.Use(WithLogging)

	r.Route(server.path, func(r chi.Router) {
		r.Get("/{linkid}", getLinkHandler)
		r.Post("/", putLinkHandler)
	})

	r.Post("/api/shorten", putLinkAPIHandler)

	log.Fatal(http.ListenAndServe(server.a, r))
}

func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		// функция Now() возвращает текущее время
		start := time.Now()

		// эндпоинт /ping
		uri := r.RequestURI
		// метод запроса
		method := r.Method

		wrappedWriter := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default status code
		}

		// точка, где выполняется хендлер pingHandler
		h.ServeHTTP(wrappedWriter, r) // обслуживание оригинального запроса

		status := wrappedWriter.statusCode

		// Since возвращает разницу во времени между start
		// и моментом вызова Since. Таким образом можно посчитать
		// время выполнения запроса.
		duration := time.Since(start)

		// отправляем сведения о запросе в zap
		sugar.Infoln(
			"status", status,
			"uri", uri,
			"method", method,
			"duration", duration,
		)

	}
	// возвращаем функционально расширенный хендлер
	return http.HandlerFunc(logFn)
}

func getLinkHandler(w http.ResponseWriter, r *http.Request) {

	// if h, ok := r.Header["Content-Type"]; !ok || h[0] != "text/plain" {
	// 	return "", errors.New("Content-Type is not text/plain")
	// }

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	if len(r.URL.Path) == 0 {
		w.WriteHeader(http.StatusForbidden)
	}

	link := chi.URLParam(r, "linkid")

	if _, ok := storage[link]; ok && len(link) != 0 {
		w.Header().Set("Location", storage[link])
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	w.WriteHeader(http.StatusNotFound)

}

func putLinkHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		return

	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if len(bs) == 0 {
		w.WriteHeader(http.StatusForbidden)
		return

	}

	var link string
	func() {

		seed := rand.New(
			rand.NewSource(time.Now().UnixNano()))

		charset := "abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, 8)
		for i := range b {
			b[i] = charset[seed.Intn(len(charset))]
		}
		link = string(b)
	}()

	var mu sync.RWMutex

	mu.Lock()
	storage[link] = string(bs)
	mu.Unlock()

	w.WriteHeader(http.StatusCreated)

	switch server.path {
	case "/":
		fmt.Fprintf(w, "%s://%s%s%s", server.scheme, server.host, server.path, link)
	default:
		fmt.Fprintf(w, "%s://%s%s%s", server.scheme, server.host, server.path+"/", link)

	}
}

func putLinkAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req raw

	json.NewDecoder(r.Body).Decode(&req)

	var link string

	func() {

		seed := rand.New(
			rand.NewSource(time.Now().UnixNano()))

		charset := "abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, 8)
		for i := range b {
			b[i] = charset[seed.Intn(len(charset))]
		}
		link = string(b)
	}()

	var mu sync.RWMutex

	mu.Lock()
	storage[link] = string(req.URL)
	mu.Unlock()

	var resp raw

	switch server.path {
	case "/":
		resp.URL = fmt.Sprintf("%s://%s%s%s", server.scheme, server.host, server.path, link)
	default:
		resp.URL = fmt.Sprintf("%s://%s%s%s", server.scheme, server.host, server.path+"/", link)

	}

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(&resp)

}
