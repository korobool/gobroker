package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

var GrossDispatcher Dispatcher
var landingTempl *template.Template

func main() {

	// TODO: Do it properly (at least call a function load templates...)
	landingTempl = template.New("valutchik.html")
	templateStr, err := ioutil.ReadFile("media/templates/valutchik.html")
	if err != nil {
		fmt.Println(err)
	}
	landingTempl, _ = landingTempl.Parse(string(templateStr))

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
