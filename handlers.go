package main

import (
	// "encoding/json"
	"fmt"
	// "io"
	// "io/ioutil"
	"net/http"
	// "strconv"

	// "github.com/gorilla/mux"
)

func GetDists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, "GetDists!\n")
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusMovedPermanently)

	// if err := json.NewEncoder(w).Encode(struct{a int, b int}{a: 5 , b: 6}) != nil {
	// 	panic(err)
	// }

	fmt.Fprint(w, "Redirect!\n")
}

func Share(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	// vars := mux.Vars(r)
	// var todoId int
	// var err error
	// if todoId, err = strconv.Atoi(vars["todoId"]); err != nil {
	// 	panic(err)
	// }
	// todo := RepoFindTodo(todoId)
	// if todo.Id > 0 {
	// 	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// 	w.WriteHeader(http.StatusOK)
	// 	if err := json.NewEncoder(w).Encode(todo); err != nil {
	// 		panic(err)
	// 	}
	// 	return
	// }

	// // If we didn't find it, 404
	// w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// w.WriteHeader(http.StatusNotFound)
	// if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
	// 	panic(err)
	// }
	fmt.Fprint(w, "Share!\n")

}

func GetLendingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	// body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	// if err != nil {
	// 	panic(err)
	// }
	// if err := r.Body.Close(); err != nil {
	// 	panic(err)
	// }
	// if err := json.Unmarshal(body, &todo); err != nil {
	// 	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// 	w.WriteHeader(422) // unprocessable entity
	// 	if err := json.NewEncoder(w).Encode(err); err != nil {
	// 		panic(err)
	// 	}
	// }

	// t := RepoCreateTodo(todo)
	// w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// w.WriteHeader(http.StatusCreated)
	// if err := json.NewEncoder(w).Encode(t); err != nil {
	// 	panic(err)
	// }
	fmt.Fprint(w, "GetLendingPage!\n")
}
