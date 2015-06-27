package main

import (
	// "encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

const (
	HashLength = 20
)

type ApiMessage struct {
	method string
	params string
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// hash := vars["hash"]
	// if len(hash) > HashLength { // TODO: add checking for alphanumeric
	// 	w.WriteHeader(http.StatusForbidden) // TODO: Provide reason message
	// 	return
	// }

	// params := ""

	// expandJson, err := GrossDispatcher.RemoteCall("expand", params, time.Second*4)

	// if err != nil {
	// 	fmt.Println("REMOTE CALL TIMEOUT")
	// }

	// w.WriteHeader(http.StatusFound)

}

func GetDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	chResult := make(chan string)

	vars := mux.Vars(r)

	appID := vars["appID"]

	apiMsg := ApiMessage{
		method: "get_app_dists",
		params: fmt.Sprintf("{\"app_id\": \"%s\"}", appID),
	}

	go GrossDispatcher.ExecuteMethod(&apiMsg, chResult)

	select {
	case result, ok := <-chResult:
		if !ok {
			fmt.Println("Chanel closed")
		}
		fmt.Fprintf(w, result)

	case <-time.After(time.Second * 4):
		w.WriteHeader(http.StatusRequestTimeout)
	}
}

func Share(w http.ResponseWriter, r *http.Request) {

}

func GetLendingPage(w http.ResponseWriter, r *http.Request) {

}
