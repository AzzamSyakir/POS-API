package controller

import (
	"encoding/json"
	"golang-api/api/responses"
	"golang-api/config"
	"net/http"
	"strings"
	"time"
)

func CreateCategories(w http.ResponseWriter, r *http.Request) {
	// Inisialisasi koneksi ke database
	var categories struct {
		CategoryID int64  `json:"categoryId"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&categories); err != nil {
		// Mengembalikan respons JSON jika gagal membaca data dari permintaan
		responses.ErrorResponse(w, "Gagal membaca data produk dari permintaan", http.StatusBadRequest)
		return
	}

	// upload gambar ke firebase dan return url yang disimpn ke var  imageURL

	// Waktu saat ini
	currentTime := time.Now()

	// Simpan produk ke database dengan menggunakan data yang telah Anda validasi
	result, err := config.DB.Exec("INSERT INTO categories ( name, created_at, updated_at) VALUES (?, ?, ?)",
		categories.Name, currentTime, currentTime)
	if err != nil {
		// Menangani kesalahan jika gagal menyimpan produk ke database
		errorMessage := "Gagal menyimpan produk ke database"
		if strings.Contains(err.Error(), "Duplicate entry") {
			errorMessage = "Produk dengan nama yang sama sudah ada."
		} else {
			errorMessage = "Terjadi kesalahan saat menyimpan produk ke database."
		}
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Mendapatkan ID yang baru saja ditambahkan ke database
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		responses.ErrorResponse(w, "Gagal mendapatkan ID produk yang baru", http.StatusInternalServerError)
		return
	}

	// Membuat objek data produk untuk dikirim dalam respons
	productData := struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}{
		ID:   lastInsertID,
		Name: categories.Name,
	}

	responses.SuccessResponse(w, "Success", productData, http.StatusCreated)
}
