package main

import (
	"fmt"
	"log"
	"net/http"
)

var dispatcher *Dispatcher

func main() {
	router := NewRouter()
	dispatcher, err := NewDispatcher("tcp://0.0.0.0:7070")
	if err != nil {
		fmt.Println(err)
	}

	// Starting zeromq loop
	go dispatcher.ZmqReadLoopRun()
	go dispatcher.ZmqWriteLoopRun()

	log.Fatal(http.ListenAndServe(":8080", router))
}
