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
		"/apps/{appID}/dists",
		GetDists,
	},
	Route{
		"Share",
		"GET",
		"/apps/{appID}/share",
		Share,
	},
	Route{
		"Redirection",
		"GET",
		"/{hash}",
		Redirect,
	},
	Route{
		"Lending",
		"GET",
		"/l/{hash}",
		GetLendingPage,
	},
}
