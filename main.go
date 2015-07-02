package main

import (
	"flag"
	"fmt"
	// "github.com/zenazn/goji"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

var rootPath string

var GrossDispatcher Dispatcher
var landingTempl *template.Template

func main() {

	verbose := flag.Bool("verbose", false, "turn the logging on")

	root := flag.String("root", "localhost:8080", "server root")
	httpBind := flag.String("httpBind", "localhost", "server HTTP bind ddress")
	httpPort := flag.Int("httpPort", 8080, "server HTTP root")
	flag.Parse()

	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}

	httpListen := fmt.Sprintf("%s:%d", *httpBind, *httpPort)
	rootPath = fmt.Sprintf("http://%s", *root)

	// TODO: Do it properly (at least call a function load templates...)
	landingTempl = template.New("valutchik.html")
	templateStr, err := ioutil.ReadFile("media/templates/valutchik.html")
	if err != nil {
		fmt.Println(err)
	}
	landingTempl, _ = landingTempl.Parse(string(templateStr))

	dispatcher, err := NewDispatcher("tcp://0.0.0.0:7070")
	if err != nil {
		fmt.Println(err)
	}
	GrossDispatcher = *dispatcher

	router := registerRoutes()

	GrossDispatcher.run()

	http.ListenAndServe(httpListen, router)
}
