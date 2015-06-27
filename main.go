package main

import (
	"fmt"
	"log"
	"net/http"
)

var GrosDispatcher Dispatcher

func main() {
	router := NewRouter()
	dispatcher, err := NewDispatcher("tcp://0.0.0.0:7070")
	GrosDispatcher = *dispatcher
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(">>>", dispatcher)

	// Starting zeromq loop
	go GrosDispatcher.ZmqReadLoopRun()
	go GrosDispatcher.ZmqWriteLoopRun()

	log.Fatal(http.ListenAndServe(":8080", router))
}
