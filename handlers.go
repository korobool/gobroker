package main

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	// "github.com/mssola/user_agent"
	//"github.com/varstr/uaparser"
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
	vars := mux.Vars(r)
	hash := vars["hash"]
	if len(hash) > HashLength { // TODO: add checking for alphanumeric
		w.WriteHeader(http.StatusForbidden) // TODO: Provide reason message
		return
	}

	msg := struct {
		Hash string `json:"hash"`
	}{hash}

	params, _ := json.Marshal(msg)

	// fmt.Println("params >> ", string(params), params, hash, msg)

	expandJson, err := GrossDispatcher.RemoteCall(
		"expand",
		string(params),
		time.Second*4,
	)

	// TODO: validate expandJson for

	var data struct {
		App_id string `json:"app_id"`
		Urls   struct {
			Android string `json:"android"`
			Apple   string `json:"apple"`
		} `json:"urls"`
	}

	err = json.Unmarshal([]byte(expandJson), &data)

	if err != nil {
		fmt.Println("REMOTE CALL TIMEOUT")
	}

	platform := getPlatform(r.UserAgent())

	if platform == PlatformAndroid {
		url := data.Urls.Android
		http.Redirect(w, r, url, http.StatusFound)

	}

	if platform == PlatformIPhone {
		url := data.Urls.Apple
		http.Redirect(w, r, url, http.StatusFound)
	}

	if platform == PlatformOther {
		w.WriteHeader(http.StatusForbidden)
	}

}

func GetDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	vars := mux.Vars(r)
	appID := vars["appID"]
	params := fmt.Sprintf("{\"app_id\": \"%s\"}", appID)

	// TODO: RemoteCall err returning wasn't checked
	expandJson, _ := GrossDispatcher.RemoteCall("get_app_dists", params, time.Second*4)
	fmt.Fprintf(w, expandJson)
}

func Share(w http.ResponseWriter, r *http.Request) {

}

func GetLendingPage(w http.ResponseWriter, r *http.Request) {

}
