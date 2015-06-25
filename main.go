package main

import (
	"log"
	"net/http"
)

var dispatcher *Dispatcher

func main() {
	router := NewRouter()
	dispatcher, _ = NewDispatcher("0.0.0.0:7070")
	log.Fatal(http.ListenAndServe(":8080", router))
}
