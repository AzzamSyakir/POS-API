package controller

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang-api/api/responses"
	"golang-api/config"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

func CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment struct {
		Name string `form:"name"`
		Type string `form:"type"`
	}

	// Mengambil nilai form-data dan mengisinya ke struct product

	payment.Name = r.FormValue("name")
	payment.Type = r.FormValue("type")

	// Validasi nilai form-data tidak boleh kosong
	if payment.Name == "" || payment.Type == "" {
		responses.ErrorResponse(w, "Semua kolom harus diisi", http.StatusBadRequest)
		return
	}

	// Mengecek apakah ada file logo yang diunggah
	file, _, err := r.FormFile("logo")
	var imageURL *string
	if err != nil {
		// Jika tidak ada file logo yang diunggah, imageURL di-set sebagai nil
		imageURL = nil
	} else {
		defer file.Close()

		// Menginisialisasi buffer untuk menyimpan logo
		var buffer bytes.Buffer
		_, err := io.Copy(&buffer, file)
		if err != nil {
			errorMessage := fmt.Sprintf("Error copying file to buffer: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		// Inisialisasi Firebase Storage client
		opt := option.WithCredentialsFile("firebase_credentials.json")
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			errorMessage := fmt.Sprintf("Error initializing app: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			errorMessage := fmt.Sprintf("Error initializing Storage client: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		// Menyimpan logo ke Firebase Storage
		bucket := os.Getenv("FIREBASE_BUCKET")
		objectName := "Logo/" + uuid.NewString()

		bucketHandle, err := client.Bucket(bucket)
		if err != nil {
			errorMessage := fmt.Sprintf("Error getting bucket handle: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		wc := bucketHandle.Object(objectName).NewWriter(context.Background())
		if _, err := io.Copy(wc, &buffer); err != nil {
			errorMessage := fmt.Sprintf("Error uploading image to Cloud Storage: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		expirationTime := time.Now().AddDate(1, 0, 0)
		googleAccessID := os.Getenv("GOOGLE_ACCESS_ID")
		PrivateKey := os.Getenv("PRIVATE_KEY")
		if err != nil {
			errorMessage := fmt.Sprintf("Error get env var: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		url, err := storage.SignedURL(bucket, objectName, &storage.SignedURLOptions{
			GoogleAccessID: googleAccessID,
			PrivateKey:     []byte(PrivateKey),
			Method:         "GET",
			Expires:        expirationTime,
		})
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to create signed URL: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}
		// akhir firebase connection
		imageURL = &url

	}
	// Waktu saat ini
	currentTime := time.Now()

	// Simpan produk ke database dengan menggunakan data yang telah Anda validasi
	result, err := config.DB.Exec("INSERT INTO payments ( name, type, logo, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		payment.Name, payment.Type, imageURL, currentTime, currentTime)
	if err != nil {
		// Menangani kesalahan jika gagal menyimpan produk ke database
		errorMessage := fmt.Sprintf("Gagal menyimpan produk ke database: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Mendapatkan ID yang baru saja ditambahkan ke database
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		errorMessage := fmt.Sprintf("Gagal mendapatkan ID produk yang baru: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Membuat objek data produk untuk dikirim dalam respons
	productData := struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
		Logo      *string   `json:"logo"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID:        lastInsertID,
		Name:      payment.Name,
		Type:      payment.Type,
		Logo:      imageURL,
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}

	responses.SuccessResponse(w, "Success", productData, http.StatusCreated)
}

func ListPayments(w http.ResponseWriter, r *http.Request) {
	type Payment struct {
		ID   int64   `json:"id"`
		Name string  `json:"name"`
		Type string  `json:"type"`
		Logo *string `json:"logo"`
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	skipStr := r.URL.Query().Get("skip")
	categoryIDStr := r.URL.Query().Get("categoryId")
	searchQuery := r.URL.Query().Get("q")

	// Convert limit and skip parameters to integers
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

	query := "SELECT id, name, type, logo FROM payments WHERE 1=1"
	args := []interface{}{} // Slice to store query parameters

	if categoryIDStr != "" {
		query += " AND (category_id = ? OR category_id IS NULL)"
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

	var payments []Payment

	for rows.Next() {
		var payment Payment
		err := rows.Scan(&payment.ID, &payment.Name, &payment.Type, &payment.Logo)
		if err != nil {
			responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payments = append(payments, payment)
	}

	total := len(payments)

	// Prepare response
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"meta": map[string]interface{}{
				"total": total,
				"limit": limit,
				"skip":  skip,
			},
			"payments": payments,
		},
	}

	// Set response headers and send the JSON response
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", response, http.StatusOK)
}
func DetailPayments(w http.ResponseWriter, r *http.Request) {
	type Payment struct {
		ID   int64   `json:"id"`
		Name string  `json:"name"`
		Type string  `json:"type"`
		Logo *string `json:"logo"`
	}

	vars := mux.Vars(r)
	PaymentId := vars["id"]

	if PaymentId == "" {
		responses.ErrorResponse(w, "ID Payment harus diisi", http.StatusBadRequest)
		return
	}

	var payment Payment

	// Menggunakan prepared statement untuk menghindari SQL Injection
	err := config.DB.QueryRow("SELECT id, name, type, logo FROM payments WHERE id=?", PaymentId).Scan(&payment.ID, &payment.Name, &payment.Type, &payment.Logo)
	if err != nil {
		if err == sql.ErrNoRows {
			errorMessage := fmt.Sprintf("payment tidak ditemukan: %v", err)

			responses.ErrorResponse(w, errorMessage, http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mengembalikan data kategori sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", payment, http.StatusOK)
}
func UpdatePayments(w http.ResponseWriter, r *http.Request) {
	paymentID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		responses.ErrorResponse(w, "ID payment tidak valid", http.StatusBadRequest)
		return
	}

	var updatedPayment struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Logo string `json:"logo"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updatedPayment); err != nil {
		errorMessage := fmt.Sprintf("Gagal membaca data payment dari permintaan: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}

	stmt, err := config.DB.Prepare("UPDATE payments SET name=?, type=?, logo=?, updated_at=NOW() WHERE id=?")
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(updatedPayment.Name, updatedPayment.Type, updatedPayment.Logo, paymentID)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paymentData := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   strconv.FormatInt(paymentID, 10),
		Name: updatedPayment.Name,
	}

	responses.SuccessResponse(w, "Success", paymentData, http.StatusOK)
}

func DeletePayments(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL pakai library mux
	vars := mux.Vars(r)
	PaymentId := vars["id"]
	if PaymentId == "" {
		responses.ErrorResponse(w, "ID payment harus disertakan", http.StatusBadRequest)
		return
	}

	// Menghapus category dari database
	_, err := config.DB.Exec("DELETE FROM payments WHERE id=?", PaymentId)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)

		return
	}

	responses.OtherResponses(w, "Success", http.StatusCreated)
}
