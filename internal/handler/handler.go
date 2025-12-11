package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"shortener/internal/config"
	"shortener/internal/config/db"
	"shortener/internal/service"
	"shortener/internal/storage"
)

type response struct {
	URL string `json:"result"`
}

type request struct {
	URL string `json:"url"`
}

type FileType struct {
	UUID        string `json:"UUID"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode      int
	wroteHeader     bool
	body            *bytes.Buffer
	isGzipped       bool
	contentEncoding string
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true

		w.contentEncoding = w.Header().Get("Content-Encoding")
		w.isGzipped = w.contentEncoding == "gzip"

		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriterWrapper) GetBody() string {
	if !w.isGzipped {
		return w.body.String()
	}

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
func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	w.body.Write(b)

	return w.ResponseWriter.Write(b)
}

func GetLinkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if len(r.URL.Path) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	link := chi.URLParam(r, "linkid")

	if storedURL, ok := storage.Read(link); ok == nil && len(link) != 0 {
		w.Header().Set("Location", storedURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func PutLinkHandler(w http.ResponseWriter, r *http.Request) {
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

	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	w.WriteHeader(http.StatusCreated)

	switch config.Server.Path {
	case "/":
		fmt.Fprintf(w, "%s://%s%s%s", config.Server.Scheme, config.Server.Host, config.Server.Path, link)
	default:
		fmt.Fprintf(w, "%s://%s%s%s", config.Server.Scheme, config.Server.Host, config.Server.Path+"/", link)
	}

	ft := service.FileType{
		UUID:        uuid.New().String(),
		ShortURL:    link,
		OriginalURL: string(bs),
	}
	service.Save(ft)
}

func PutLinkAPIHandler(w http.ResponseWriter, r *http.Request) {
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

	if err != nil || mt != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var resp response
	switch config.Server.Path {
	case "/":
		resp.URL = fmt.Sprintf("%s://%s%s%s", config.Server.Scheme, config.Server.Host, config.Server.Path, link)
	default:
		resp.URL = fmt.Sprintf("%s://%s%s%s", config.Server.Scheme, config.Server.Host, config.Server.Path+"/", link)
	}

	ft := service.FileType{
		UUID:        uuid.New().String(),
		ShortURL:    link,
		OriginalURL: req.URL,
	}

	service.Save(ft)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&resp)
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

		if len(body) > 1000 {
			body = body[:1000] + "...[truncated]"
		}

		config.Sugar.Infoln(
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

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
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

func DBHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !db.Ping() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}
