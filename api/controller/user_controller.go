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

// mengambil user berdasarkan ID.
func GetUser(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		// Tangani jika ID pengguna tidak ada
		responses.ErrorResponse(w, "ID pengguna harus diisi", http.StatusBadRequest)
		return
	}

	// Mengambil pengguna dari database berdasarkan ID
	var (
		id       int
		name     string
		email    string
		password string
	)

	err := config.DB.QueryRow("SELECT id, name, email, password FROM users WHERE id=?", userID).Scan(&id, &name, &email, &password)
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "user tidak ditemukan", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		// Anda dapat menambahkan lebih banyak data pengguna sesuai kebutuhan
	}{
		Name:     name,
		Email:    email,
		Password: password,
	}

	// Mengembalikan data pengguna sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", userData, http.StatusCreated)
}

// Fetch User
func FetchUser(w http.ResponseWriter, r *http.Request) {
	// Mengambil pengguna dari database berdasarkan ID
	var (
		id       int
		username string
		email    string
		password string
	)
	err := config.DB.QueryRow("SELECT * FROM users").Scan(&id, &username, &email, &password)
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
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		// Anda dapat menambahkan lebih banyak data pengguna sesuai kebutuhan
	}{
		Username: username,
		Email:    email,
		Password: password,
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
	_, err = config.DB.Exec("UPDATE users SET name=?, email=?, password=? WHERE id=?", updatedUser.Name, updatedUser.Email, hashedPassword, userID) // Mengganti userID menjadi updatedUser.Name
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Membuat objek data pengguna untuk dikirim dalam respons
	userData := struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		// Anda dapat menambahkan lebih banyak data pengguna sesuai kebutuhan
	}{
		Name:     updatedUser.Name,
		Email:    updatedUser.Email,
		Password: string(hashedPassword),
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
