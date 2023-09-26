package routes

import (
	"golang-api/api/controller"
	"golang-api/config"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Auth api
	r.HandleFunc("/users/login", controller.LoginUser).Methods("POST")
	// Users api
	r.HandleFunc("/users", controller.CreateUser).Methods("POST")
	r.HandleFunc("/users/{id}", controller.UpdateUser).Methods("PUT")
	r.HandleFunc("/users/{id}", controller.DeleteUser).Methods("DELETE")
	r.HandleFunc("/users/{id}", controller.GetUser).Methods("GET")
	r.HandleFunc("/users", controller.FetchUser).Methods("GET")
	// Products api
	r.HandleFunc("/products", controller.CreateProduct).Methods("POST")

	// categories api
	r.HandleFunc("/categories", controller.CreateCategories).Methods("POST")

	return r
}

func RunServer() {
	config.InitDB()
	router := SetupRoutes()

	// Mulai server HTTP dengan router yang telah dikonfigurasi
	http.Handle("/", router)
	http.ListenAndServe(":9000", nil)

}
