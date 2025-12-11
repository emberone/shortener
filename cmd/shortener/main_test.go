package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"shortener/internal/handler"

	"github.com/stretchr/testify/assert"
)

func TestGetLinkHandler(t *testing.T) {

	testCasesGetLinkHandler := []struct {
		method       string
		expectedCode int
		expectedBody string
	}{
		{method: http.MethodGet, expectedCode: http.StatusNotFound, expectedBody: ""},
		{method: http.MethodPost, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
	}

	for _, tc := range testCasesGetLinkHandler {
		t.Run(tc.method, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/", nil)
			w := httptest.NewRecorder()

			// вызовем хендлер как обычную функцию, без запуска самого сервера
			handler.GetLinkHandler(w, r)

			assert.Equal(t, tc.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")
		})
	}

}

func TestPostLinkHandler(t *testing.T) {

	testCasesPutLinkHandler := []struct {
		method       string
		expectedCode int
		sentBody     string
	}{
		{method: http.MethodGet, expectedCode: http.StatusMethodNotAllowed},
		{method: http.MethodPost, expectedCode: http.StatusCreated, sentBody: "https://practicum.yandex.ru/"},
	}

	for _, tc := range testCasesPutLinkHandler {
		t.Run(tc.method, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/", bytes.NewBufferString(tc.sentBody))
			r.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			handler.PutLinkHandler(w, r)

			assert.Equal(t, tc.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")
		})
	}
}
