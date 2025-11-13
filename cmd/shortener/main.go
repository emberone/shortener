package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
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
type fileType struct {
	UUID        string `json:"UUID"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var storage map[string]string = map[string]string{}

var server struct {
	a       string
	b       string
	path    string
	host    string
	scheme  string
	storage string
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

		w.contentEncoding = w.Header().Get("Content-Encoding")
		w.isGzipped = w.contentEncoding == "gzip"

		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	w.body.Write(b)

	return w.ResponseWriter.Write(b)
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
	flag.StringVar(&server.storage, "f", "storage.db", "storage")
	flag.Parse()

	if a := os.Getenv("SERVER_ADDRESS"); a != "" {
		server.a = a
	}
	if b := os.Getenv("SERVER_ADDRESS"); b != "" {
		server.b = b
	}
	if f := os.Getenv("FILE_STORAGE_PATH"); f != "" {
		server.storage = f
	}

	fmt.Printf("os.Environ(): %v\n", os.Environ())

	readFromFile()

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

func saveToFile(ft fileType) {

	_, err := os.Stat(server.storage)
	if err != nil {
		f, err := os.Create(server.storage)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		defer f.Close()

		f.Write([]byte("[\n]"))
	}

	f, err := os.OpenFile(server.storage, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	defer f.Close()

	_, err = f.Seek(-1, io.SeekEnd)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	var bs [1]byte
	f.Read(bs[:])

	stat, err := os.Stat(server.storage)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	if string(bs[0]) == "]" {
		f.Truncate(stat.Size() - 1)
	}

	var mu sync.Mutex
	mu.Lock()
	//err = json.NewEncoder(f).Encode(ft)
	fmt.Fprintf(f, `    {"uuid":"%s","short_url":"%s","original_url":"%s"},`+"\n]", ft.UUID, ft.ShortURL, ft.OriginalURL)
	mu.Unlock()

}

func readFromFile() {

	f, err := os.OpenFile(server.storage, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	bs, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	bs = bs[1 : len(bs)-1]

	ss := strings.Split(strings.TrimSpace(string(bs)), "\n")

	for i := range ss {

		s := strings.TrimSpace(strings.ReplaceAll(ss[i], "},", "}"))
		var ft fileType

		err := json.Unmarshal([]byte(s), &ft)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}

		storage[ft.ShortURL] = ft.OriginalURL

	}

	fmt.Printf("storage: %v\n", storage)
}
