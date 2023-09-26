package controller

import (
	"database/sql"
	"encoding/json"
	"golang-api/api/responses"
	"golang-api/config"
	"net/http"
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

	// Mendapatkan username dan password dari data pengguna
	username, ok := user["username"].(string)
	if !ok {
		responses.ErrorResponse(w, "Username harus diisi", http.StatusBadRequest)
		return
	}

	password, ok := user["password"].(string)
	if !ok {
		responses.ErrorResponse(w, "Password harus diisi", http.StatusBadRequest)
		return
	}

	// Mengecek apakah pengguna ada di database dan mengambil password dari database
	var dbPassword string
	err := config.DB.QueryRow("SELECT password FROM users WHERE username=?", username).Scan(&dbPassword)
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
	claims["username"] = username
	claims["exp"] = time.Now().Add(time.Hour * 1).Unix() // Token berlaku selama 1 jam

	// Menandatangani token dengan secret key (gantilah dengan secret key yang kuat)
	secretKey := []byte("secretKey") // Ganti dengan secret key yang lebih kuat
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		responses.ErrorResponse(w, "Gagal membuat token JWT", http.StatusInternalServerError)
		return
	}

	// Mengembalikan token dan pesan sukses
	response := map[string]interface{}{"token": tokenString}
	responses.SuccessResponse(w, "succes", response, http.StatusCreated)
}
