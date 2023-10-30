package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-api/api/responses"
	"golang-api/config"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

func ListCategories(w http.ResponseWriter, r *http.Request) {
	type Category struct {
		ID       int64     `json:"id"`
		Name     string    `json:"name"`
		Category *Category `json:"category"`
	}

	limitStr := r.URL.Query().Get("limit")
	skipStr := r.URL.Query().Get("skip")
	categoryIDStr := r.URL.Query().Get("categoryId")
	searchQuery := r.URL.Query().Get("q")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		responses.ErrorResponse(w, "Invalid 'limit' parameter", http.StatusBadRequest)
		return
	}

	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		responses.ErrorResponse(w, "Invalid 'skip' parameter", http.StatusBadRequest)
		return
	}

	query := "SELECT id, name FROM categories WHERE 1=1"
	var args []interface{}

	if categoryIDStr != "" {
		query += " AND (id = ? OR id IS NULL)"
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			responses.ErrorResponse(w, "Invalid 'categoryId' parameter", http.StatusBadRequest)
			return
		}
		args = append(args, categoryID)
	}

	if searchQuery != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+searchQuery+"%")
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, skip)

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category

	for rows.Next() {
		var category Category
		err := rows.Scan(&category.ID, &category.Name)
		if err != nil {
			responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}

		categories = append(categories, category)
	}

	total := len(categories)

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"meta": map[string]interface{}{
				"total": total,
				"limit": limit,
				"skip":  skip,
			},
			"categories": categories,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", response, http.StatusOK)
}
func DetailCategories(w http.ResponseWriter, r *http.Request) {
	type Category struct {
		ID       int64     `json:"id"`
		Name     string    `json:"name"`
		Category *Category `json:"category,omitempty"`
	}
	vars := mux.Vars(r)
	categoryID := vars["id"]

	if categoryID == "" {
		responses.ErrorResponse(w, "ID Category harus diisi", http.StatusBadRequest)
		return
	}

	var category Category

	// Menggunakan prepared statement untuk menghindari SQL Injection
	err := config.DB.QueryRow("SELECT id, name FROM categories WHERE id=?", categoryID).Scan(&category.ID, &category.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			errorMessage := fmt.Sprintf("Category tidak ditemukan: %v", err)

			responses.ErrorResponse(w, errorMessage, http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mengembalikan data kategori sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", category, http.StatusOK)
}
func UpdateCategories(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID kategori dari parameter menggunakan mux
	vars := mux.Vars(r)
	categoryID := vars["id"]
	if categoryID == "" {
		responses.ErrorResponse(w, "ID kategori harus disertakan", http.StatusBadRequest)
		return
	}

	// Mendapatkan data kategori dari body permintaan
	var updatedCategories struct {
		ID   int64  `json:"id"` // Ganti 'id' dengan 'ID'
		Name string `json:"name"`
		// Anda dapat menambahkan lebih banyak field kategori sesuai kebutuhan
	}

	if err := json.NewDecoder(r.Body).Decode(&updatedCategories); err != nil {
		errorMessage := fmt.Sprintf("Gagal membaca data kategori dari permintaan: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Menggunakan prepared statement untuk menghindari SQL Injection
	stmt, err := config.DB.Prepare("UPDATE categories SET name=?, updated_at=NOW() WHERE id=?")
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Memperbarui kategori di database
	_, err = stmt.Exec(updatedCategories.Name, categoryID)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Membuat objek data kategori untuk dikirim dalam respons
	categoryData := struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		UpdatedAt string `json:"updated_at"`
	}{
		ID:        updatedCategories.ID,
		Name:      updatedCategories.Name,
		UpdatedAt: time.Now().Format(time.RFC3339), // Menggunakan format waktu yang sesuai
	}

	responses.SuccessResponse(w, "Kategori berhasil diperbarui", categoryData, http.StatusOK)
}
func DeleteCategories(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL pakai library mux
	vars := mux.Vars(r)
	CategoryId := vars["id"]
	if CategoryId == "" {
		responses.ErrorResponse(w, "ID Category harus disertakan", http.StatusBadRequest)
		return
	}

	// Menghapus category dari database
	_, err := config.DB.Exec("DELETE FROM categories WHERE id=?", CategoryId)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)

		return
	}

	responses.OtherResponses(w, "Success", http.StatusCreated)
}
