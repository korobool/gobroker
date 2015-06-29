package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

const (
	HashLength     = 20
	AppIdLength    = 15
	DefaultTimeout = time.Second * 1
	ApiKeyHeader   = "X-API-KEY"
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

func checkAuth(authHeader string) (string, error) {
	msg := struct {
		ApiKey string `json:"api_key"`
	}{authHeader}

	params, _ := json.Marshal(msg)

	customerJSON, err := GrossDispatcher.RemoteCall(
		"get_customer",
		string(params),
		DefaultTimeout,
	)
	if err != nil {
		return "", err
	}

	var data struct {
		CustomerId string `json:"customer_id"`
	}

	err = json.Unmarshal([]byte(customerJSON), &data)
	if data.CustomerId == "" {
		return "", errors.New("No such API key")
	}

	return data.CustomerId, nil
}

func checkAppId(appId string, customerId string) error {
	msg := struct {
		CustomerId string `json:"customer_id"`
	}{customerId}

	params, _ := json.Marshal(msg)

	appsJSON, err := GrossDispatcher.RemoteCall(
		"get_customer_apps",
		string(params),
		DefaultTimeout,
	)

	var data struct {
		Apps []string `json:"apps"`
	}

	err = json.Unmarshal([]byte(appsJSON), &data)
	if err != nil {
		return err
	}

	for _, value := range data.Apps {
		if value == appId {
			return nil
		}
	}
	return errors.New("Not permited AppId for this customer")
}

func GetDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	customerId, err := checkAuth(r.Header.Get(ApiKeyHeader))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	appId := vars["appID"]

	if len(appId) > AppIdLength {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = checkAppId(appId, customerId)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//params := fmt.Sprintf("{\"app_id\": \"%s\"}", appId)
	msg := struct {
		AppId string `json:"app_id"`
	}{appId}

	params, _ := json.Marshal(msg)

	// TODO: RemoteCall err returning wasn't checked
	distsJSON, _ := GrossDispatcher.RemoteCall("get_app_dists", string(params), DefaultTimeout)

	fmt.Fprintf(w, distsJSON)
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

	// TODO: Refactor copy-paste
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

	err = landingTempl.ExecuteTemplate(w, landingResult.Template, context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
