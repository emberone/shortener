package main

import (
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

var storage map[string]string = map[string]string{}

func main() {

	r := chi.NewRouter()

	r.Get("/", getLinkHandler)
	r.Post("/", putLinkHandler)

	http.ListenAndServe(":8080", r)
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

	if _, ok := storage[r.URL.Path[1:]]; !ok {
		w.WriteHeader(http.StatusNotFound)

	}

	w.Write([]byte(storage[r.URL.Path[1:]]))

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

}
