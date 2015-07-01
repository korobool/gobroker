package main

import (
	"fmt"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	// "net/http"
)

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc web.HandlerFunc
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
		"/:hash",
		Redirect,
	},
	Route{
		"GET",
		"/l/:hash",
		Landing,
	},
}

func registerRoutes() {

	for _, route := range routes {
		var handler web.HandlerFunc //http.Handler
		handler = route.HandlerFunc

		if route.Method == "GET" {
			goji.Get(route.Pattern, handler)
		}

		if route.Method == "POST" {
			goji.Post(route.Pattern, handler)
		}
		fmt.Println("Registered", route)
	}

}
