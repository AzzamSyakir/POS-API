package controller

import (
	"database/sql"
	"encoding/json"
	"golang-api/api/responses"
	"golang-api/config"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
	// Inisialisasi koneksi ke database
	var user struct {
		Id       int64  // Gunakan tipe data int64 untuk ID
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		// Mengembalikan respons JSON jika gagal membaca data dari permintaan
		responses.ErrorResponse(w, "Gagal membaca data pengguna dari permintaan", http.StatusBadRequest)
		return
	}

	// Hashing password sebelum disimpan ke database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		responses.ErrorResponse(w, "Gagal melakukan hashing password", http.StatusInternalServerError)
		return
	}
	// Waktu saat ini
	currentTime := time.Now()

	// Simpan pengguna ke database dengan menggunakan data yang telah Anda validasi
	result, err := config.DB.Exec("INSERT INTO users (name, email, password, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		user.Name, user.Email, hashedPassword, currentTime, currentTime) // Mengganti user.Username menjadi user.Name
	if err != nil {
		// Menangani kesalahan jika gagal menyimpan pengguna ke database
		errorMessage := "Gagal menyimpan pengguna ke database"
		if strings.Contains(err.Error(), "Duplicate entry") {
			errorMessage = "Email sudah digunakan. Silakan gunakan email lain."
		} else {
			errorMessage = "Terjadi kesalahan saat menyimpan pengguna ke database."
		}
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Mendapatkan ID yang baru saja ditambahkan ke database
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		responses.ErrorResponse(w, "Gagal mendapatkan ID pengguna yang baru", http.StatusInternalServerError)
		return
	}

	// Mengisi ID pada struct user dengan nilai yang baru saja didapatkan
	user.Id = lastInsertId

	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Id        int64     `json:"id"` // Gunakan ID yang telah diisi
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		Id:        user.Id,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}

	responses.SuccessResponse(w, "Success", userData, http.StatusCreated)
}

// Mengambil user berdasarkan ID.
func GetUser(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL
	vars := mux.Vars(r)
	userID, ok := vars["id"]

	if !ok || userID == "" {
		// Tangani jika ID pengguna tidak ada atau kosong
		responses.ErrorResponse(w, "ID pengguna harus diisi", http.StatusBadRequest)
		return
	}

	// Mengambil data pengguna dari database berdasarkan ID
	var (
		name       string
		email      string
		created_at string
		updated_at string
	)

	// Query database menggunakan prepared statement untuk menghindari SQL injection
	err := config.DB.QueryRow("SELECT name, email, created_at, updated_at FROM users WHERE id=?", userID).
		Scan(&name, &email, &created_at, &updated_at)
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "User tidak ditemukan", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Created_at string `json:"created_at"`
		Updated_at string `json:"updated_at"`
	}{
		Name:       name,
		Email:      email,
		Created_at: created_at,
		Updated_at: updated_at,
	}

	// Mengembalikan data pengguna sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", userData, http.StatusOK)
}

// Fetch User
func FetchUser(w http.ResponseWriter, r *http.Request) {
	var (
		id         int
		username   string
		email      string
		password   string
		created_at string
		updated_at string
	)
	err := config.DB.QueryRow("SELECT * FROM users").Scan(&id, &username, &email, &password, &created_at, &updated_at)
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "tidak ada data dalam tabel", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Username   string `json:"username"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		Created_at string `json:"created_at"`
		Updated_at string `json:"updated_at"`
		// Anda dapat menambahkan lebih banyak data pengguna sesuai kebutuhan
	}{
		Username:   username,
		Email:      email,
		Password:   password,
		Created_at: created_at,
		Updated_at: updated_at,
	}
	// Mengembalikan data pengguna sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", userData, http.StatusCreated)
}

// update user
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter pakai library mux
	vars := mux.Vars(r)
	userID := vars["id"]
	if userID == "" {
		responses.ErrorResponse(w, "ID pengguna harus disertakan", http.StatusBadRequest)
		return
	}

	// Mendapatkan data pengguna dari body permintaan
	var updatedUser struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		responses.ErrorResponse(w, "Gagal membaca data pengguna dari permintaan", http.StatusBadRequest)
		return
	}

	// Hashing password sebelum disimpan ke database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedUser.Password), bcrypt.DefaultCost)
	if err != nil {
		responses.ErrorResponse(w, "Gagal melakukan hashing password", http.StatusInternalServerError)
		return
	}

	// Memperbarui pengguna di database
	_, err = config.DB.Exec("UPDATE users SET name=?, email=?, password=?,  updated_at=NOW()  WHERE id=?", updatedUser.Name, updatedUser.Email, hashedPassword, userID) // Mengganti userID menjadi updatedUser.Name
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		UpdatedAt string `json:"updated_at"`
		// Anda dapat menambahkan lebih banyak data pengguna sesuai kebutuhan
	}{
		Name:      updatedUser.Name,
		Email:     updatedUser.Email,
		Password:  string(hashedPassword),
		UpdatedAt: time.Now().Format(time.RFC3339), // Menggunakan format waktu yang sesuai

	}

	responses.SuccessResponse(w, "Success", userData, http.StatusOK)
}

// delete
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL pakai library mux
	vars := mux.Vars(r)
	userID := vars["id"]
	if userID == "" {
		responses.ErrorResponse(w, "ID pengguna harus disertakan", http.StatusBadRequest)
		return
	}

	// Menghapus pengguna dari database
	_, err := config.DB.Exec("DELETE FROM users WHERE id=?", userID)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)

		return
	}

	responses.OtherResponses(w, "Success", http.StatusCreated)
}
