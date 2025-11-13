package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

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

	ft := fileType{
		Uuid:         uuid.New().String(),
		Short_url:    link,
		Original_url: string(bs),
	}
	saveToFile(ft)
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

	ft := fileType{
		Uuid:         uuid.New().String(),
		Short_url:    link,
		Original_url: req.URL,
	}
	saveToFile(ft)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&resp)
}
