package controller

import (
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang-api/api/responses"
	"golang-api/config"

	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

func ListProducts(w http.ResponseWriter, r *http.Request) {
	// Membaca nilai parameter dari query string
	limitStr := r.URL.Query().Get("limit")
	skipStr := r.URL.Query().Get("skip")
	categoryIDStr := r.URL.Query().Get("categoryId")
	searchQuery := r.URL.Query().Get("q")

	// Mengonversi nilai "limit" dan "skip" menjadi integer
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		responses.ErrorResponse(w, "Parameter 'limit' tidak valid", http.StatusBadRequest)
		return
	}

	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		responses.ErrorResponse(w, "Parameter 'skip' tidak valid", http.StatusBadRequest)
		return
	}

	// Query SQL dasar untuk mengambil data produk dari tabel "products"
	query := "SELECT id, sku, name, stock, price, image, category_id, created_at, updated_at FROM products"

	// Variabel untuk menyimpan argumen untuk query
	args := []interface{}{}

	// Menggunakan filter berdasarkan category ID jika diberikan
	if categoryIDStr != "" {
		query += " AND category_id = ?"
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			responses.ErrorResponse(w, "Parameter 'categoryId' tidak valid", http.StatusBadRequest)
			return
		}
		args = append(args, categoryID)
	}

	if searchQuery != "" {
		// Sesuaikan dengan kolom yang sesuai untuk pencarian
		query += " AND name LIKE ?"
		args = append(args, "%"+searchQuery+"%")
	}

	// Menambahkan LIMIT dan OFFSET ke query
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, skip)

	// Eksekusi query menggunakan config.DB.Query
	rows, err := config.DB.Query(query, args...)
	if err != nil {
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Define the struct DataProduct
	type DataProduct struct {
		ID         int    `json:"id"`
		SKU        string `json:"sku"`
		Name       string `json:"name"`
		Stock      int    `json:"stock"`
		Price      int    `json:"price"`
		Image      string `json:"image"`
		CategoryID int    `json:"category_id"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}

	// Create a slice to store the query results
	var products []DataProduct

	// Fetch data from the query results
	for rows.Next() {
		var product DataProduct
		err := rows.Scan(&product.ID, &product.SKU, &product.Name, &product.Stock, &product.Price, &product.Image, &product.CategoryID, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		products = append(products, product)
	}
	// Menghitung total data
	total := len(products)

	response := map[string]interface{}{
		"success": true,
		"message": "Success",
		"data": map[string]interface{}{
			"meta": map[string]interface{}{
				"total": total,
				"limit": limit,
				"skip":  skip,
			},
			"products": products,
		},
	}

	// Mengembalikan response JSON
	responses.SuccessResponse(w, "succes", response, http.StatusCreated)
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Inisialisasi koneksi ke database
	var product struct {
		CategoryID int64  `form:"categoryId"`
		Name       string `form:"name"`
		Price      string `form:"price"`
		Stock      string `form:"stock"`
	}

	// Mengambil nilai form-data dan mengisinya ke struct product
	product.CategoryID, _ = strconv.ParseInt(r.FormValue("categoryId"), 10, 64)
	product.Name = r.FormValue("name")
	product.Price = r.FormValue("price")
	product.Stock = r.FormValue("stock")
	// upload gambar ke firebase dan return url yang disimpn ke var  imageURL

	// Mendapatkan file gambar dari permintaan HTTP
	file, _, err := r.FormFile("image")
	if err != nil {
		responses.ErrorResponse(w, "gagal baca image", http.StatusBadRequest)
	}
	//firebase
	FirebaseConfig := &firebase.Config{
		StorageBucket: "firebase_credentials.json",
	}
	opt := option.WithCredentialsFile("firebase_credentials.json")
	app, err := firebase.NewApp(context.Background(), FirebaseConfig, opt)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Storage(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	bucket, err := client.DefaultBucket()
	if err != nil {
		log.Fatalln(err)
	}
	// Generate nama unik untuk file gambar (misalnya, dengan UUID)
	uniqueName := uuid.New().String()
	// Upload file gambar ke Firebase Storage
	obj := bucket.Object(uniqueName)
	wc := obj.NewWriter(context.Background())
	defer wc.Close()

	// upload gmbr
	_, err = io.Copy(wc, file)
	if err != nil {
		responses.ErrorResponse(w, "gagal upload", http.StatusBadRequest)
	}

	// Dapatkan URL gambar yang diunggah
	imageURL := "https://storage.googleapis.com/nama-bucket-firebase.appspot.com/" + uniqueName

	// Waktu saat ini
	currentTime := time.Now()

	// Simpan produk ke database dengan menggunakan data yang telah Anda validasi
	result, err := config.DB.Exec("INSERT INTO products (category_id, name, price, stock, image, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		product.CategoryID, product.Name, product.Price, product.Stock, imageURL, currentTime, currentTime)
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
		ID         int64     `json:"id"`
		CategoryID int64     `json:"categoryId"`
		Name       string    `json:"name"`
		Price      string    `json:"price"`
		Stock      string    `json:"stock"`
		Image      string    `json:"image"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}{
		ID:         lastInsertID,
		CategoryID: product.CategoryID,
		Name:       product.Name,
		Price:      product.Price,
		Stock:      product.Stock,
		Image:      imageURL,
		CreatedAt:  currentTime,
		UpdatedAt:  currentTime,
	}

	responses.SuccessResponse(w, "Success", productData, http.StatusCreated)
}
