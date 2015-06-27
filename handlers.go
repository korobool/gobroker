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
		params: fmt.Sprintf("{'appID': %s}", appID),
	}

	fmt.Println(">>>>>>>>>>>>>>>>!!!", GrosDispatcher)
	go GrosDispatcher.ExecuteMethod(&apiMsg, chResult)

	select {
	case result, ok := <-chResult:
		if !ok {
			fmt.Println("Chanel closed")
		}
		fmt.Println(result)
	case <-time.After(time.Second * 3):
		fmt.Println("timeout")
	}
}

func Share(w http.ResponseWriter, r *http.Request) {

}

func GetLendingPage(w http.ResponseWriter, r *http.Request) {

}
