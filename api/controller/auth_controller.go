package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-api/api/responses"
	"golang-api/config"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

func LoginUser(w http.ResponseWriter, r *http.Request) {
	var user map[string]interface{}

	// Membaca data JSON dari body permintaan
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		responses.ErrorResponse(w, "Gagal membaca data pengguna dari permintaan", http.StatusBadRequest)
		return
	}

	// Mendapatkan email dan password dari data pengguna
	email, ok := user["email"].(string)
	if !ok {
		responses.ErrorResponse(w, "Email harus diisi", http.StatusBadRequest)
		return
	}

	password, ok := user["password"].(string)
	if !ok {
		responses.ErrorResponse(w, "Password harus diisi", http.StatusBadRequest)
		return
	}

	// Mengecek apakah pengguna ada di database dan mengambil ID dan password dari database
	var userID int
	var dbPassword string
	var dbName string
	err := config.DB.QueryRow("SELECT id, name, password FROM users WHERE email=?", email).Scan(&userID, &dbName, &dbPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "User tidak ditemukan", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Membandingkan password yang dimasukkan dengan password yang ada di database
	if err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)); err != nil {
		responses.ErrorResponse(w, "Password salah", http.StatusUnauthorized)
		return
	}

	// Jika login berhasil, buat token JWT
	token := jwt.New(jwt.SigningMethodHS256)

	// Menentukan klaim (claims) token
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = userID
	claims["username"] = dbName
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // Token berlaku selama 24 jam

	// Menandatangani token dengan secret key
	secretKeyString := os.Getenv("SECRET_KEY")
	secretKey := []byte(secretKeyString)

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		errorMessage := fmt.Sprintf("Gagal membuat token JWT: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Mengembalikan token dan pesan sukses
	response := map[string]interface{}{"token": tokenString}
	responses.SuccessResponse(w, "success", response, http.StatusCreated)
}
