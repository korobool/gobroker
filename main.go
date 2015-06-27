package main

import (
	"fmt"
	"log"
	"net/http"
)

var GrossDispatcher Dispatcher

func main() {
	router := NewRouter()
	dispatcher, err := NewDispatcher("tcp://0.0.0.0:7070")
	if err != nil {
		fmt.Println(err)
	}
	GrossDispatcher = *dispatcher

	// // Starting zeromq loop
	// go GrossDispatcher.ZmqReadLoopRun()
	// go GrossDispatcher.ZmqWriteLoopRun()
	GrossDispatcher.run()

	log.Fatal(http.ListenAndServe(":8080", router))
}
