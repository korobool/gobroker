package main

import (
	"encoding/json"
	"errors"
	"fmt"
	// "github.com/gorilla/mux"
	// "github.com/zenazn/goji"
	"github.com/julienschmidt/httprouter"
	// "github.com/zenazn/goji/web"
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

func Redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// hash := c.URLParams["hash"]
	hash := ps.ByName("hash")

	if len(hash) > HashLength { // TODO: add checking for alphanumeric
		w.WriteHeader(http.StatusForbidden) // TODO: Provide reason message
		return
	}

	msg := struct {
		Hash string `json:"hash"`
	}{hash}

	params, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expandJSON, err := GrossDispatcher.RemoteCall("expand", params, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	// TODO: validate expandJSON for
	var expandData struct {
		AppId string `json:"app_id"`
		Urls  struct {
			Android string `json:"android"`
			Apple   string `json:"apple"`
		} `json:"urls"`
	}

	err = json.Unmarshal(expandJSON, &expandData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if expandData.AppId == "" {
		http.NotFound(w, r)
		return
	}

	platform := getPlatform(r.UserAgent())

	statistics := StatMessage{
		AppId:    expandData.AppId,
		Link:     r.RequestURI,
		IP:       strings.Split(r.RemoteAddr, ":")[0],
		UA:       r.UserAgent(),
		Time:     time.Now().Format(time.RFC3339),
		Hash:     hash,
		Platform: getDeviceType(platform),
		LinkType: "redirect",
	}

	params, _ = json.Marshal(statistics)

	_, err = GrossDispatcher.RemoteCall("open_commit", params, DefaultTimeout)
	if err != nil {
		fmt.Println(err)
	}

	if platform == PlatformAndroid {
		url := expandData.Urls.Android
		http.Redirect(w, r, url, http.StatusFound)
	} else if platform == PlatformIPhone {
		url := expandData.Urls.Apple
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

	customerJSON, err := GrossDispatcher.RemoteCall("get_customer", params, DefaultTimeout)
	if err != nil {
		return "", err
	}

	var resultData struct {
		CustomerId string `json:"customer_id"`
	}

	err = json.Unmarshal(customerJSON, &resultData)

	if resultData.CustomerId == "" {
		return "", errors.New("No such API key")
	}

	return resultData.CustomerId, nil
}

func checkAppId(appId string, customerId string) error {

	msg := struct {
		CustomerId string `json:"customer_id"`
	}{customerId}

	params, _ := json.Marshal(msg)

	appsJSON, err := GrossDispatcher.RemoteCall("get_customer_apps", params, DefaultTimeout)
	if err != nil {
		return err
	}

	var dataResult struct {
		Apps []string `json:"apps"`
	}

	err = json.Unmarshal(appsJSON, &dataResult)
	if err != nil {
		return err
	}

	for _, app := range dataResult.Apps {
		if app == appId {
			return nil
		}
	}
	return errors.New("Not permited AppId for this customer")
}

func AppDists(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	customerId, err := checkAuth(r.Header.Get(ApiKeyHeader))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// appId := c.URLParams["appId"]
	appId := ps.ByName("appId")

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

	distsJSON, err := GrossDispatcher.RemoteCall("get_app_dists", params, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}
	fmt.Fprintf(w, string(distsJSON))
}

func AppShare(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	customerId, err := checkAuth(r.Header.Get(ApiKeyHeader))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// vars := c.URLParams
	appId := ps.ByName("appId")

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

	infoJSON, err := GrossDispatcher.RemoteCall("get_app_info", params, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	var infoResult struct {
		UrlApple   string `json:"url_apple"`
		UrlAndroid string `json:"url_android"`
	}

	err = json.Unmarshal(infoJSON, &infoResult)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shortenData := struct {
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
		r.Host,
		struct {
			Apple   string `json:"apple"`
			Android string `json:"android"`
		}{infoResult.UrlApple, infoResult.UrlAndroid},
	}

	params, _ = json.Marshal(shortenData)

	shortenJSON, err := GrossDispatcher.RemoteCall("shorten", params, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	var hashData struct {
		Hash string `json:"hash"`
	}

	err = json.Unmarshal(shortenJSON, &hashData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resultData := struct {
		Redirect string `json:"redirect"`
		Landing  string `json:"landing"`
	}{
		fmt.Sprintf("%s/%s", rootPath, hashData.Hash),
		fmt.Sprintf("%s/l/%s", rootPath, hashData.Hash),
	}
	resultJSON, _ := json.Marshal(resultData)

	fmt.Fprintf(w, string(resultJSON))
}

func Landing(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// hash := c.URLParams["hash"]

	hash := ps.ByName("hash")
	if len(hash) > HashLength { // TODO: add checking for alphanumeric
		w.WriteHeader(http.StatusForbidden) // TODO: Provide reason message
		return
	}

	msg := struct {
		Hash string `json:"hash"`
	}{hash}

	params, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expandJson, err := GrossDispatcher.RemoteCall("expand", params, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	// TODO: validate expandJson for
	var expandData struct {
		AppId string `json:"app_id"`
		Urls  struct {
			Android string `json:"android"`
			Apple   string `json:"apple"`
		} `json:"urls"`
	}

	err = json.Unmarshal(expandJson, &expandData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if expandData.AppId == "" {
		http.NotFound(w, r)
		return
	}

	// TODO: Refactor copy-paste
	/////////////////////////////////////////////////////////////////////////////

	landingMsg := struct {
		AppId string `json:"app_id"`
	}{expandData.AppId}

	landingParams, _ := json.Marshal(landingMsg)

	landigJSON, err := GrossDispatcher.RemoteCall("get_app_landing", landingParams, DefaultTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	var landingResult struct {
		MetaApple   string `json:"meta_apple"`
		MetaAndroid string `json:"meta_android"`
		Template    string `json:"template"`
	}

	err = json.Unmarshal(landigJSON, &landingResult)

	platform := getPlatform(r.UserAgent())

	var meta map[string]string

	var context struct {
		Meta  map[string]string
		Image string
	}

	if platform == PlatformAndroid {

		json.Unmarshal([]byte(landingResult.MetaAndroid), &meta)

		context.Meta = meta
		context.Image = meta["image"]

	} else if platform == PlatformIPhone {
		json.Unmarshal([]byte(landingResult.MetaApple), &meta)

		context.Meta = meta
		context.Image = meta["image"]

	} else {
		json.Unmarshal([]byte(landingResult.MetaAndroid), &meta)

		context.Meta = nil
		context.Image = meta["image"]
	}

	err = landingTempl.ExecuteTemplate(w, landingResult.Template, context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	statistics := StatMessage{
		AppId:    expandData.AppId,
		Link:     r.RequestURI,
		IP:       strings.Split(r.RemoteAddr, ":")[0],
		UA:       r.UserAgent(),
		Time:     time.Now().Format(time.RFC3339),
		Hash:     hash,
		Platform: getDeviceType(platform),
		LinkType: "landing",
	}

	params, _ = json.Marshal(statistics)

	_, err = GrossDispatcher.RemoteCall("open_commit", params, DefaultTimeout)
	if err != nil {
		fmt.Println(err)
	}

}
