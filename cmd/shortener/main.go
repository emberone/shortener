package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var storage map[string]string = map[string]string{}

func main() {

	http.HandleFunc("/", mainPage)

	http.ListenAndServe(":8080", nil)
}

func mainPage(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		var link string
		var err error

		if link, err = putLink(r); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Printf("err: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/" + link))

	case http.MethodGet:

		var link string
		var err error

		if link, err = getLink(r); err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Printf("err: %v\n", err)
			return
		}

		w.Header().Set("Location", link)
		w.WriteHeader(http.StatusTemporaryRedirect)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func putLink(r *http.Request) (string, error) {

	if h, ok := r.Header["Content-Type"]; !ok || h[0] != "text/plain" {
		return "", errors.New("Content-Type is not text/plain")
	}

	bs, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	if len(bs) == 0 {
		return "", errors.New("body is empty")
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

	return link, nil

}

func getLink(r *http.Request) (string, error) {

	// if h, ok := r.Header["Content-Type"]; !ok || h[0] != "text/plain" {
	// 	return "", errors.New("Content-Type is not text/plain")
	// }

	if len(r.URL.Path) == 0 {
		return "", errors.New("no url path has been provided")
	}

	if _, ok := storage[r.URL.Path[1:]]; !ok {
		fmt.Printf("r.URL.Path[1:]: %v\n", r.URL.Path[1:])
		return "", errors.New("not found in storage")
	}

	return storage[r.URL.Path[1:]], nil
}
