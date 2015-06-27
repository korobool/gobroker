package main

import (
	// "encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type ApiMessage struct {
	method string
	params string
}

func Redirect(w http.ResponseWriter, r *http.Request) {

}

func GetDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusMovedPermanently)

	chResult := make(chan string)

	vars := mux.Vars(r)

	appID := vars["appID"]

	apiMsg := ApiMessage{
		method: "get_app_dists",
		params: fmt.Sprintf("{\"app_id\": \"%s\"}", appID),
	}

	fmt.Println(">>>>>>>>>>>>>>>>!!!", GrosDispatcher)
	go GrosDispatcher.ExecuteMethod(&apiMsg, chResult)

	select {
	case result, ok := <-chResult:
		if !ok {
			fmt.Println("Chanel closed")
		}
		fmt.Printf("GetDists handler result: %s", result)
	case <-time.After(time.Second * 10):
		fmt.Println("timeout")
		// write here
		return
	}
}

func Share(w http.ResponseWriter, r *http.Request) {

}

func GetLendingPage(w http.ResponseWriter, r *http.Request) {

}
