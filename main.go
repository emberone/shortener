package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var cars = map[string]string{
	"id1": "Renault Logan",
	"id2": "Renault Duster",
	"id3": "BMW X6",
	"id4": "BMW M5",
	"id5": "VW Passat",
	"id6": "VW Jetta",
	"id7": "Audi A4",
	"id8": "Audi Q7",
}

// modelFunc — вспомогательная функция для вывода определённой машины.
func modelFunc(id string) string {

	for i := 1; i != len(cars); i++ {

		if c, ok := cars[id+strconv.Itoa(i)]; ok {
			fmt.Printf("c: %v\n", c)
			//return c
		}
	}

	return "unknown identifier " + id
}

func modelHandle(rw http.ResponseWriter, r *http.Request) {
	carID := chi.URLParam(r, "id")
	if carID == "" {
		http.Error(rw, "carID param is missed", http.StatusBadRequest)
		return
	}
	rw.Write([]byte(modelFunc(carID)))
}

// carFunc — вспомогательная функция для вывода определённой машины.
func brandFunc(id string) string {

	id = strings.ToLower(id)

	list := []string{}

	for i := 1; i != len(cars)+1; i++ {

		if c, ok := cars["id"+strconv.Itoa(i)]; ok {

			if strings.Split(c, " ")[0] == id {
				list = append(list, strings.Split(c, " ")[1])
			}
		}
	}

	return strings.Join(list, ", ")
}

func brandHandle(rw http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brand")
	if brandID == "" {
		http.Error(rw, "brandID param is missed", http.StatusBadRequest)
		return
	}
	rw.Write([]byte(brandFunc(brandID)))
}

func main() {

	r := chi.NewRouter()

	r.Use(middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Route("/cars", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("cars"))
		})
		r.Route("/{brand}", func(r chi.Router) {
			r.Get("/", brandHandle)
			r.Get("/{model}", modelHandle)
		})

	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
