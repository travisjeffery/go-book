package main

import (
	"log"
	nethttp "net/http"

	"github.com/gorilla/mux"
	http "github.com/travisjeffery/httplog/http"
)

func main() {
	srv := http.NewServer()
	r := mux.NewRouter()
	r.HandleFunc("/", srv.Produce).Methods("POST")
	r.HandleFunc("/", srv.Consume).Methods("GET")
	nethttp.Handle("/", r)
	log.Fatal(nethttp.ListenAndServe(":8080", nil))
}
