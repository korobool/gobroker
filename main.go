package main

import (
	"flag"
	"fmt"
	"github.com/zenazn/goji"
	"html/template"
	"io/ioutil"
)

var rootPath string

var GrossDispatcher Dispatcher
var landingTempl *template.Template

func main() {
	root := flag.String("root", "localhost:8080", "server root")
	flag.Parse()

	rootPath = fmt.Sprintf("http://%s", *root)

	// TODO: Do it properly (at least call a function load templates...)
	landingTempl = template.New("valutchik.html")
	templateStr, err := ioutil.ReadFile("media/templates/valutchik.html")
	if err != nil {
		fmt.Println(err)
	}
	landingTempl, _ = landingTempl.Parse(string(templateStr))

	flag.Set("bind", ":8080")
	registerRoutes()

	dispatcher, err := NewDispatcher("tcp://0.0.0.0:7070")
	if err != nil {
		fmt.Println(err)
	}
	GrossDispatcher = *dispatcher

	GrossDispatcher.run()
	goji.Serve()

}
