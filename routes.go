package main

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"Dists",
		"GET",
		"/apps/{appId}/dists",
		AppDists,
	},
	Route{
		"Share",
		"GET",
		"/apps/{appId}/share",
		AppShare,
	},
	Route{
		"Redirection",
		"GET",
		"/{hash}",
		Redirect,
	},
	Route{
		"Landing",
		"GET",
		"/l/{hash}",
		Landing,
	},
}
