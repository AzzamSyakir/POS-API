package routes

import (
	"golang-api/api/controller"
	"golang-api/api/middleware"
	"golang-api/config"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Rute yang tidak memerlukan otentikasi
	r.HandleFunc("/users", controller.CreateUser).Methods("POST")
	r.HandleFunc("/users/login", controller.LoginUser).Methods("POST")

	// Router untuk rute dengan dua middleware
	protectedRoutes := r.PathPrefix("/").Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware)

	// Rute yang dilindungi oleh middleware
	// Users API
	protectedRoutes.HandleFunc("/users/{id}", controller.UpdateUser).Methods("PUT")
	protectedRoutes.HandleFunc("/users/{id}", controller.DeleteUser).Methods("DELETE")
	protectedRoutes.HandleFunc("/users/{id}", controller.GetUser).Methods("GET")
	protectedRoutes.HandleFunc("/users", controller.FetchUser).Methods("GET")

	// Products API
	protectedRoutes.HandleFunc("/products", controller.CreateProduct).Methods("POST")
	protectedRoutes.HandleFunc("/products", controller.ListProducts).Methods("GET")
	protectedRoutes.HandleFunc("/products/{id}", controller.DetailProducts).Methods("GET")
	protectedRoutes.HandleFunc("/products/{id}", controller.UpdateProducts).Methods("PUT")
	protectedRoutes.HandleFunc("/products/{id}", controller.DeleteProducts).Methods("DELETE")

	// Categories API
	protectedRoutes.HandleFunc("/categories", controller.CreateCategories).Methods("POST")
	protectedRoutes.HandleFunc("/categories", controller.ListCategories).Methods("GET")
	protectedRoutes.HandleFunc("/categories/{id}", controller.DetailCategories).Methods("GET")
	protectedRoutes.HandleFunc("/categories/{id}", controller.UpdateCategories).Methods("PUT")
	protectedRoutes.HandleFunc("/categories/{id}", controller.DeleteCategories).Methods("DELETE")

	// Payments API
	protectedRoutes.HandleFunc("/payments", controller.CreatePayment).Methods("POST")
	protectedRoutes.HandleFunc("/payments", controller.ListPayments).Methods("GET")
	protectedRoutes.HandleFunc("/payments/{id}", controller.DetailPayments).Methods("GET")
	protectedRoutes.HandleFunc("/payments/{id}", controller.UpdatePayments).Methods("PUT")
	protectedRoutes.HandleFunc("/payments/{id}", controller.DeletePayments).Methods("DELETE")
	// Orders API
	protectedRoutes.HandleFunc("/orders", controller.CreateOrders).Methods("POST")
	protectedRoutes.HandleFunc("/orders", controller.ListOrders).Methods("GET")
	protectedRoutes.HandleFunc("/payments/{id}", controller.DetailPayments).Methods("GET")
	protectedRoutes.HandleFunc("/payments/{id}", controller.UpdatePayments).Methods("PUT")
	protectedRoutes.HandleFunc("/payments/{id}", controller.DeletePayments).Methods("DELETE")

	return r
}

func RunServer() {
	config.InitDB()
	router := SetupRoutes()

	// Mulai server HTTP dengan router yang telah dikonfigurasi
	http.Handle("/", router)
	http.ListenAndServe(":9000", nil)

}
