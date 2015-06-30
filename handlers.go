package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	//"strconv"
	"strings"
	"time"
)

const (
	HashLength     = 20
	AppIdLength    = 15
	DefaultTimeout = time.Second * 1
	ApiKeyHeader   = "X-API-KEY"
	DomainHeader   = "X-Host"
)

type ApiMessage struct {
	method string
	params string
}

type StatMessage struct {
	AppId    string `json:"app_id"`
	Link     string `json:"link"`
	IP       string `json:"ip"`
	UA       string `json:"user_agent"`
	Time     string `json:"time"`
	Hash     string `json:"hash"`
	Platform string `json:"platform"`
	LinkType string `json:"link_type"`
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
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
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

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
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if data.AppId == "" {
		http.NotFound(w, r)
		return
	}

	platform := getPlatform(r.UserAgent())

	statistics := StatMessage{
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

	_, err = GrossDispatcher.RemoteCall("open_commit", string(params), DefaultTimeout)
	if err != nil {
		fmt.Println(err)
	}

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
		//WORKAROUND: recieving customer_id as int from Auth-worker
		CustomerId string `json:"customer_id"`
	}

	err = json.Unmarshal([]byte(customerJSON), &data)

	//WORKAROUND: recieving customer_id as int from Auth-worker
	//customer := strconv.Itoa(data.CustomerId)
	if data.CustomerId == "" {
		return "", errors.New("No such API key")
	}

	return data.CustomerId, nil
}

func checkAppId(appId string, customerId string) error {
	//WORKAROUND: recieving customer_id as int from Auth-worker
	//customer, _ := strconv.Atoi(customerId)

	fmt.Println("customerId>>>>>", customerId)

	msg := struct {
		CustomerId string `json:"customer_id"`
		//WORKAROUND: recieving customer_id as int from Auth-worker
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

func AppDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	customerId, err := checkAuth(r.Header.Get(ApiKeyHeader))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	appId := vars["appId"]

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

func AppShare(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	customerId, err := checkAuth(r.Header.Get(ApiKeyHeader))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	appId := vars["appId"]

	if len(appId) > AppIdLength {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = checkAppId(appId, customerId)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	msg := struct {
		AppId string `json:"app_id"`
	}{appId}

	params, _ := json.Marshal(msg)

	infoJSON, _ := GrossDispatcher.RemoteCall("get_app_info", string(params), DefaultTimeout)

	type InfoResult struct {
		UrlApple   string `json:"url_apple"`
		UrlAndroid string `json:"url_android"`
	}
	var infoResult InfoResult

	err = json.Unmarshal([]byte(infoJSON), &infoResult)

	data := struct {
		CustomerId string `json:"customer_id"`
		AppId      string `json:"app_id"`
		Domain     string `json:"domain"`
		Urls       struct {
			Apple   string `json:"apple"`
			Android string `json:"android"`
		} `json:"urls"`
	}{
		customerId,
		appId,
		r.Header.Get(DomainHeader),
		struct {
			Apple   string `json:"apple"`
			Android string `json:"android"`
		}{infoResult.UrlApple, infoResult.UrlAndroid},
	}

	params, _ = json.Marshal(data)

	shortenJSON, _ := GrossDispatcher.RemoteCall("shorten", string(params), DefaultTimeout)

	var shortenData struct {
		Hash string `json:"hash"`
	}

	err = json.Unmarshal([]byte(shortenJSON), &shortenData)

	resultData := struct {
		Redirect string `json:"redirect"`
		Landing  string `json:"landing"`
	}{
		fmt.Sprintf("%s/%s", rootPath, shortenData.Hash),
		fmt.Sprintf("%s/l/%s", rootPath, shortenData.Hash),
	}
	resultJSON, _ := json.Marshal(resultData)

	fmt.Fprintf(w, string(resultJSON))
}

func Landing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

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

	statistics := StatMessage{
		AppId:    data.AppId,
		Link:     r.RequestURI,
		IP:       strings.Split(r.RemoteAddr, ":")[0],
		UA:       r.UserAgent(),
		Time:     time.Now().Format(time.RFC3339),
		Hash:     hash,
		Platform: getDeviceType(platform),
		LinkType: "landing",
	}

	params, _ = json.Marshal(statistics)

	GrossDispatcher.RemoteCall("open_commit", string(params), DefaultTimeout)
}
