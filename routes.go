package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	// "github.com/zenazn/goji"
	// "github.com/zenazn/goji/web"

	// "net/http"
)

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc httprouter.Handle
}

type Routes []Route

var routes = Routes{
	Route{
		"GET",
		"/apps/:appId/dists",
		AppDists,
	},
	Route{
		"GET",
		"/apps/:appId/share",
		AppShare,
	},
	Route{
		"GET",
		"/l/:hash",
		Landing,
	},
	// Route{
	// 	"GET",
	// 	"/:hash",
	// 	Redirect,
	// },
}

func registerRoutes() *httprouter.Router {
	router := httprouter.New()

	for _, route := range routes {
		var handler httprouter.Handle //http.Handler
		handler = route.HandlerFunc

		if route.Method == "GET" {
			router.GET(route.Pattern, handler)
		}

		if route.Method == "POST" {
			router.POST(route.Pattern, handler)
		}
		fmt.Println("Registered", route)
	}
	return router

}
