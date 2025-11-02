package main

import (
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
)

var storage map[string]string = map[string]string{}

var server struct {
	a      string
	b      string
	path   string
	host   string
	scheme string
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

	r := chi.NewRouter()
	r.Route(server.path, func(r chi.Router) {
		r.Get("/{linkid}", getLinkHandler)
		r.Post("/", putLinkHandler)
	})

	log.Fatal(http.ListenAndServe(server.a, r))
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
