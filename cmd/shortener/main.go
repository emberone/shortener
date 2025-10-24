package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

var storage map[string]string = map[string]string{}

var server string

func main() {

	flag.StringVar(&server, "a", "localhost:8080", "server address and port")
	flag.Parse()

	r := chi.NewRouter()

	r.Get("/{linkid}", getLinkHandler)
	r.Post("/", putLinkHandler)

	fmt.Printf("server: %v\n", server)
	http.ListenAndServe(server, r)
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
	}

	if h, ok := r.Header["Content-Type"]; !ok || h[0] != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
	}

	if len(bs) == 0 {
		w.WriteHeader(http.StatusForbidden)

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

	storage[link] = string(bs)

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "http://%s/%s", server, link)

}
