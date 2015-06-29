package main

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	// "github.com/mssola/user_agent"
	//"github.com/varstr/uaparser"
	//"html/template"
	//"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	HashLength     = 20
	DefaultTimeout = time.Second * 1
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

	expandJSON, err := GrossDispatcher.RemoteCall(
		"expand",
		string(params),
		DefaultTimeout,
	)

	// TODO: validate expandJSON for

	var data struct {
		AppId string `json:"app_id"`
		Urls  struct {
			Android string `json:"android"`
			Apple   string `json:"apple"`
		} `json:"urls"`
	}

	err = json.Unmarshal([]byte(expandJSON), &data)

	if err != nil {
		fmt.Println("REMOTE CALL TIMEOUT")
	}

	if data.AppId == "" {
		// w.WriteHeader(http.NotFound(w, r)) // TODO: add "not found" page
		http.NotFound(w, r)
		return
	}

	platform := getPlatform(r.UserAgent())

	statistics := struct {
		AppId    string `json:"app_id"`
		Link     string `json:"link"`
		IP       string `json:"ip"`
		UA       string `json:"user_agent"`
		Time     string `json:"time"`
		Hash     string `json:"hash"`
		Platform string `json:"platform"`
		LinkType string `json:"link_type"`
	}{
		AppId:    data.AppId,
		Link:     r.RequestURI,
		IP:       strings.Split(r.RemoteAddr, ":")[0],
		UA:       r.UserAgent(),
		Time:     time.Now().Format(time.RFC3339),
		Hash:     hash,
		Platform: getDeviceType(platform),
		LinkType: "redirect",
	}

	params, _ = json.Marshal(statistics)

	GrossDispatcher.RemoteCall("open_commit", string(params), DefaultTimeout)

	if platform == PlatformAndroid {
		url := data.Urls.Android
		http.Redirect(w, r, url, http.StatusFound)
	} else if platform == PlatformIPhone {
		url := data.Urls.Apple
		http.Redirect(w, r, url, http.StatusFound)
	} else {
		w.WriteHeader(http.StatusForbidden) // TODO: add forbidden page
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
		DefaultTimeout,
	)

	// TODO: validate expandJson for

	var data struct {
		AppId string `json:"app_id"`
		Urls  struct {
			Android string `json:"android"`
			Apple   string `json:"apple"`
		} `json:"urls"`
	}

	err = json.Unmarshal([]byte(expandJson), &data)

	if err != nil {
		fmt.Println("REMOTE CALL TIMEOUT")
	}

	if data.AppId == "" {
		// w.WriteHeader(http.NotFound(w, r)) // TODO: add "not found" page
		http.NotFound(w, r)
		return
	}

	/////////////////////////////////////////////////////////////////////////////

	landingMsg := struct {
		AppId string `json:"app_id"`
	}{data.AppId}

	landingParams, _ := json.Marshal(landingMsg)

	landigJSON, err := GrossDispatcher.RemoteCall("get_app_landing", string(landingParams), DefaultTimeout)

	var landingResult struct {
		MetaApple   string `json:"meta_apple"`
		MetaAndroid string `json:"meta_android"`
		Template    string `json:"template"`
	}

	err = json.Unmarshal([]byte(landigJSON), &landingResult)

	platform := getPlatform(r.UserAgent())

	var meta map[string]string

	var context struct {
		Meta  map[string]string
		Image string
	}

	if platform == PlatformAndroid {

		err = json.Unmarshal([]byte(landingResult.MetaAndroid), &meta)

		context.Meta = meta
		context.Image = meta["image"]

	} else if platform == PlatformIPhone {
		err = json.Unmarshal([]byte(landingResult.MetaApple), &meta)

		context.Meta = meta
		context.Image = meta["image"]

	} else {
		err = json.Unmarshal([]byte(landingResult.MetaAndroid), &meta)

		context.Meta = nil
		context.Image = meta["image"]
	}

	//t := template.New(landingResult.Template)

	// templateString, _ := ioutil.ReadFile(
	// 	"media/templates/valutchik.tpl")

	// t, _ = t.Parse(string(templateString))

	err = landingTempl.ExecuteTemplate(w, landingResult.Template, context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
