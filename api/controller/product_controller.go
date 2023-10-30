package controller

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang-api/api/responses"
	"golang-api/config"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

// list producs
func ListProducts(w http.ResponseWriter, r *http.Request) {
	type Product struct {
		ID        int64  `json:"id"`
		SKU       string `json:"sku"`
		Name      string `json:"name"`
		Stock     string `json:"stock"`
		Price     string `json:"price"`
		Image     string `json:"image"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Category  *struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"category"`
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

	// SQL query with JOIN to fetch data from "products" and "categories" tables
	query := `
	    SELECT 
		p.id, p.sku, p.name, p.stock, p.price, p.image, 
		p.created_at, p.updated_at,
		c.id AS category_id, c.name AS category_name
	    FROM products p
	    LEFT JOIN categories c ON p.category_id = c.id
	    WHERE 1=1
	`

	args := []interface{}{} // Slice to store query parameters

	if categoryIDStr != "" {
		query += " AND (p.category_id = ? OR p.category_id IS NULL)"
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			responses.ErrorResponse(w, "Invalid 'categoryId' parameter", http.StatusBadRequest)
			return
		}
		args = append(args, categoryID)
	}

	if searchQuery != "" {
		query += " AND p.name LIKE ?"
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

	var products []Product

	for rows.Next() {
		var product Product
		var categoryID sql.NullInt64
		var categoryName sql.NullString
		err := rows.Scan(&product.ID, &product.SKU, &product.Name, &product.Stock, &product.Price, &product.Image, &product.CreatedAt, &product.UpdatedAt, &categoryID, &categoryName)
		if err != nil {
			responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if categoryID.Valid {
			product.Category = &struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			}{}
			product.Category.ID = categoryID.Int64
			product.Category.Name = categoryName.String
		}

		products = append(products, product)
	}

	total := len(products)

	// Prepare response
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"meta": map[string]interface{}{
				"total": total,
				"limit": limit,
				"skip":  skip,
			},
			"products": products,
		},
	}

	// Set response headers and send the JSON response
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", response, http.StatusOK)
}

// create products
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var product struct {
		CategoryID *int64 `form:"categoryId"`
		Name       string `form:"name"`
		Price      string `form:"price"`
		Stock      string `form:"stock"`
	}

	categoryID, err := strconv.ParseInt(r.FormValue("categoryId"), 10, 64)
	if err != nil {
		errorMessage := fmt.Sprintf("Invalid categoryId: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}

	product.CategoryID = &categoryID
	product.Name = r.FormValue("name")
	product.Price = r.FormValue("price")
	product.Stock = r.FormValue("stock")

	if product.Name == "" || product.Price == "" || product.Stock == "" {
		responses.ErrorResponse(w, "Semua kolom harus diisi", http.StatusBadRequest)
		return
	}

	var count int
	config.DB.QueryRow("SELECT COUNT(*) FROM categories WHERE id = ?", categoryID).Scan(&count)

	if count == 0 {
		product.CategoryID = nil
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		errorMessage := fmt.Sprintf("No file uploaded: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		responses.ErrorResponse(w, "Error copying file to buffer", http.StatusInternalServerError)
		return
	}

	opt := option.WithCredentialsFile("firebase_credentials.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		responses.ErrorResponse(w, "Error initializing app", http.StatusInternalServerError)
		return
	}

	client, err := app.Storage(context.Background())
	if err != nil {
		responses.ErrorResponse(w, "Error initializing Storage client", http.StatusInternalServerError)
		return
	}

	bucket := os.Getenv("FIREBASE_BUCKET")
	objectName := "Product/" + uuid.NewString()

	bucketHandle, err := client.Bucket(bucket)
	if err != nil {
		responses.ErrorResponse(w, "Error getting bucket handle", http.StatusInternalServerError)
		return
	}

	wc := bucketHandle.Object(objectName).NewWriter(context.Background())
	if _, err := io.Copy(wc, &buffer); err != nil {
		responses.ErrorResponse(w, "Error uploading image to Cloud Storage", http.StatusInternalServerError)
		return
	}
	if err := wc.Close(); err != nil {
		responses.ErrorResponse(w, "Error closing writer", http.StatusInternalServerError)
		return
	}

	storageClient, err := storage.NewClient(context.Background(), opt)
	if err != nil {
		responses.ErrorResponse(w, "Error creating storage client", http.StatusInternalServerError)
		return
	}
	defer storageClient.Close()

	object := storageClient.Bucket(bucket).Object(objectName)
	_, err = object.Attrs(context.Background())
	if err != nil {
		responses.ErrorResponse(w, "Error getting object attributes", http.StatusInternalServerError)
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

	imageURL, err := storage.SignedURL(bucket, objectName, &storage.SignedURLOptions{
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

	currentTime := time.Now()
	firstLetter := strings.ToUpper(string(product.Name[0]))
	rand.Seed(time.Now().UnixNano())
	randomDigits := rand.Intn(900) + 100
	SKU := fmt.Sprintf("%s%d", firstLetter, randomDigits)

	result, err := config.DB.Exec("INSERT INTO products (category_id, name, sku, price, stock, image, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		product.CategoryID, product.Name, SKU, product.Price, product.Stock, imageURL, currentTime, currentTime)
	if err != nil {
		responses.ErrorResponse(w, "Gagal menyimpan produk ke database", http.StatusInternalServerError)
		return
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		responses.ErrorResponse(w, "Gagal mendapatkan ID produk yang baru", http.StatusInternalServerError)
		return
	}

	productData := struct {
		ID        int64       `json:"id"`
		SKU       string      `json:"sku"`
		Name      string      `json:"name"`
		Stock     string      `json:"stock"`
		Price     string      `json:"price"`
		Image     string      `json:"image"`
		Category  interface{} `json:"category"`
		CreatedAt time.Time   `json:"created_at"`
		UpdatedAt time.Time   `json:"updated_at"`
	}{
		ID:        lastInsertID,
		SKU:       SKU,
		Name:      product.Name,
		Stock:     product.Stock,
		Price:     product.Price,
		Image:     imageURL,
		Category:  product.CategoryID,
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}

	responses.SuccessResponse(w, "Success", productData, http.StatusCreated)
}

// DetailProducts
func DetailProducts(w http.ResponseWriter, r *http.Request) {
	type Product struct {
		ID        int64  `json:"id"`
		SKU       string `json:"sku"`
		Name      string `json:"name"`
		Stock     string `json:"stock"`
		Price     string `json:"price"`
		Image     string `json:"image"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Category  *struct {
			ID   *int64  `json:"id"`
			Name *string `json:"name"`
		} `json:"category"`
	}

	// get id param
	vars := mux.Vars(r)
	productID := vars["id"]

	if productID == "" {
		// Tangani jika ID produk tidak ada
		responses.ErrorResponse(w, "ID produk harus diisi", http.StatusBadRequest)
		return
	}

	var product Product

	// get data product from db using id that passed from param
	err := config.DB.QueryRow("SELECT id, sku, name, stock, price, image, created_at, updated_at FROM products WHERE id=?", productID).Scan(&product.ID, &product.SKU, &product.Name, &product.Stock, &product.Price, &product.Image, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "Produk tidak ditemukan", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mengembalikan data produk sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", product, http.StatusOK)
}

// UpdateProducts menghandle permintaan untuk memperbarui data produk berdasarkan ID produk
func UpdateProducts(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID produk dari parameter menggunakan mux
	vars := mux.Vars(r)
	productID := vars["id"]
	if productID == "" {
		responses.ErrorResponse(w, "ID produk harus disertakan", http.StatusBadRequest)
		return
	}
	// Mendapatkan data produk dari body permintaan
	var updatedProduct struct {
		Name       string `json:"name"`
		SKU        string `json:"sku"`
		Stock      int    `json:"stock"`
		Price      int    `json:"price"`
		Image      string `json:"image"`
		CategoryID *int64 `json:"category_id"`
		// Anda dapat menambahkan lebih banyak field produk sesuai kebutuhan
	}
	if err := json.NewDecoder(r.Body).Decode(&updatedProduct); err != nil {
		errorMessage := fmt.Sprintf("Gagal membaca data produk dari permintaan: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Validasi categoryId
	var count int
	err := config.DB.QueryRow("SELECT COUNT(*) FROM categories WHERE id = ?", updatedProduct.CategoryID).Scan(&count)
	if err != nil {
		log.Printf("Error checking categoryID validity: %v\n", err)
		errorMessage := fmt.Sprintf("Error checking categoryID validity: %v", err)

		// Handle error, misalnya dengan mengirim respons error ke client
		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}

	// Jika categoryId tidak valid, set CategoryID menjadi nil
	if count == 0 {
		updatedProduct.CategoryID = nil
	}

	// Menggunakan prepared statement untuk menghindari SQL Injection
	stmt, err := config.DB.Prepare("UPDATE products SET name=?, sku=?, stock=?, price=?, image=?, category_id=?, updated_at=NOW() WHERE id=?")
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Memperbarui produk di database, termasuk field image, category_id, dan updated_at
	_, err = stmt.Exec(updatedProduct.Name, updatedProduct.SKU, updatedProduct.Stock, updatedProduct.Price, updatedProduct.Image, updatedProduct.CategoryID, productID)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Membuat objek data produk untuk dikirim dalam respons
	productData := struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		SKU        string `json:"sku"`
		Stock      int    `json:"stock"`
		Price      int    `json:"price"`
		Image      string `json:"image"`
		CategoryID *int64 `json:"category_id"`
		UpdatedAt  string `json:"updated_at"`
	}{
		ID:         productID,
		Name:       updatedProduct.Name,
		SKU:        updatedProduct.SKU,
		Stock:      updatedProduct.Stock,
		Price:      updatedProduct.Price,
		Image:      updatedProduct.Image,
		CategoryID: updatedProduct.CategoryID,
		UpdatedAt:  time.Now().Format(time.RFC3339), // Menggunakan format waktu yang sesuai
	}

	responses.SuccessResponse(w, "Produk berhasil diperbarui", productData, http.StatusOK)
}

// delete product dengan id
func DeleteProducts(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID pengguna dari parameter URL pakai library mux
	vars := mux.Vars(r)
	userID := vars["id"]
	if userID == "" {
		responses.ErrorResponse(w, "ID products harus disertakan", http.StatusBadRequest)
		return
	}

	// Menghapus pengguna dari database
	_, err := config.DB.Exec("DELETE FROM products WHERE id=?", userID)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)

		return
	}

	responses.OtherResponses(w, "Success", http.StatusCreated)
}
