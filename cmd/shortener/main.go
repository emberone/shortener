package main

import (
	"bytes"
	"compress/gzip"
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
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type request struct {
	URL string `json:"url"`
}

type response struct {
	URL string `json:"result"`
}

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
	statusCode      int
	wroteHeader     bool
	body            *bytes.Buffer
	isGzipped       bool
	contentEncoding string
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true

		// Check if response will be gzipped
		w.contentEncoding = w.Header().Get("Content-Encoding")
		w.isGzipped = w.contentEncoding == "gzip"

		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	// Write to our buffer
	w.body.Write(b)

	// Write to the original response writer
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) GetBody() string {
	if !w.isGzipped {
		return w.body.String()
	}

	// Decompress gzipped body for logging
	reader, err := gzip.NewReader(w.body)
	if err != nil {
		return fmt.Sprintf("[gzip decompression error: %v]", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Sprintf("[gzip read error: %v]", err)
	}

	return string(decompressed)
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
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
		panic(err)
	}
	defer logger.Sync()

	sugar = *logger.Sugar()

	r := chi.NewRouter()
	r.Use(WithLogging, gzipHandle)

	r.Route(server.path, func(r chi.Router) {
		r.Get("/{linkid}", getLinkHandler)
		r.Post("/", putLinkHandler)
	})

	r.Post("/api/shorten", putLinkAPIHandler)

	log.Fatal(http.ListenAndServe(server.a, r))
}

func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.RequestURI
		method := r.Method

		wrappedWriter := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}

		h.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)
		body := wrappedWriter.GetBody()

		// Truncate long responses for cleaner logs
		if len(body) > 1000 {
			body = body[:1000] + "...[truncated]"
		}

		sugar.Infoln(
			"status", wrappedWriter.statusCode,
			"uri", uri,
			"method", method,
			"content_encoding", wrappedWriter.contentEncoding,
			"response_size", wrappedWriter.body.Len(),
			"decompressed_size", len(body),
			"body", body,
			"duration", duration,
		)
	}
	return http.HandlerFunc(logFn)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle incoming gzip requests
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// Handle outgoing gzip responses
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// Check if we should compress this response
			// For now, let's compress text-based responses
			contentType := w.Header().Get("Content-Type")
			shouldCompress := strings.Contains(contentType, "text/") ||
				strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "application/javascript")

			if shouldCompress {
				gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
				if err != nil {
					io.WriteString(w, err.Error())
					return
				}
				defer gz.Close()

				w.Header().Set("Content-Encoding", "gzip")
				next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func getLinkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if len(r.URL.Path) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	link := chi.URLParam(r, "linkid")

	if storedURL, ok := storage[link]; ok && len(link) != 0 {
		w.Header().Set("Location", storedURL)
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(bs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var link string
	func() {
		seed := rand.New(rand.NewSource(time.Now().UnixNano()))
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

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var link string
	func() {
		seed := rand.New(rand.NewSource(time.Now().UnixNano()))
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
	storage[link] = req.URL
	mu.Unlock()

	var resp response
	switch server.path {
	case "/":
		resp.URL = fmt.Sprintf("%s://%s%s%s", server.scheme, server.host, server.path, link)
	default:
		resp.URL = fmt.Sprintf("%s://%s%s%s", server.scheme, server.host, server.path+"/", link)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&resp)
}
