package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-api/api/responses"
	"golang-api/config"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func CreateOrders(w http.ResponseWriter, r *http.Request) {
	// declared struct
	type Payments struct {
		Id   int    `json:"payment_id"`
		Name string `json:"name"`
		Type string `json:"type"`
		Logo string `json:"logo"`
	}

	type OrderProduct struct {
		Id          int64 `json:"id"`
		Order_id    int64 `json:"order_id"`
		ProductID   int   `json:"product_id"`
		Qty         int   `json:"qty"`
		Total_price int   `json:"total_price"`
	}

	type CreateOrderRequest struct {
		PaymentID int            `json:"payment_id"`
		TotalPaid int            `json:"total_paid"`
		Products  []OrderProduct `json:"products"`
	}

	// create var to handle request from body
	var request CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		errorMessage := fmt.Sprintf("Gagal membaca data produk dari permintaan: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
		return
	}
	// decode token dan ambil data dari token
	// declare struct untuk simpan data dari token
	type DataUsers struct {
		Username string `json:"username"`
		UserId   int    `json:"user_id"`
		jwt.StandardClaims
	}
	// Ambil secret key dari environment variable
	secretKeyString := os.Getenv("SECRET_KEY")
	secretKey := []byte(secretKeyString)

	auth := r.Header.Get("Authorization")
	if auth == "" {
		responses.ErrorResponse(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	splitToken := strings.Split(auth, "Bearer ")
	if len(splitToken) < 2 {
		responses.ErrorResponse(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
		return
	}
	auth = splitToken[1]

	// Mendekode token dengan klaim DataUsers
	token, err := jwt.ParseWithClaims(auth, &DataUsers{}, func(token *jwt.Token) (interface{}, error) {
		// Periksa metode tanda tangan token (optional)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		errorMessage := fmt.Sprintf("Token is invalid or expired: %v", err)
		responses.ErrorResponse(w, errorMessage, http.StatusUnauthorized)
		return
	}

	// Mengekstrak klaim dari token
	claims, ok := token.Claims.(*DataUsers)
	if !ok {
		responses.ErrorResponse(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
		return
	}
	// Mengekstrak username dan user_id dari klaim dan mencetaknya
	username := claims.Username
	userID := claims.UserId

	// Membuat slice untuk menampung hasil query

	// ambil data payment dari db
	var payment Payments //create var to handle data from db
	// get payment id from request
	paymentID := request.PaymentID
	payment.Id = paymentID

	// Lakukan query ke database
	err = config.DB.QueryRow("SELECT name, type, logo FROM payments WHERE id=?", paymentID).Scan(&payment.Name, &payment.Type, &payment.Logo)
	// Periksa apakah ada kesalahan dalam query
	if err != nil {
		if err == sql.ErrNoRows {
			responses.ErrorResponse(w, "Payment dengan ID "+strconv.Itoa(paymentID)+" tidak ditemukan", http.StatusNotFound)
			return
		}
		responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Menghitung total harga untuk pesanan dan memeriksa stok produk
	var total_price int

	for i, orderProduct := range request.Products {
		productID := orderProduct.ProductID
		var stock, price int
		err := config.DB.QueryRow("SELECT stock, price FROM products WHERE id=?", productID).Scan(&stock, &price)
		if err != nil {
			responses.ErrorResponse(w, "Produk dengan ID "+strconv.Itoa(productID)+" tidak ditemukan", http.StatusNotFound)
			return
		}

		if orderProduct.Qty > stock {
			responses.ErrorResponse(w, "Stok produk dengan ID "+strconv.Itoa(productID)+" tidak mencukupi", http.StatusConflict)
			return
		}

		// Menghitung total harga produk berdasarkan kuantitas dan harga dari database
		totalPrice := orderProduct.Qty * price

		// Menyimpan total harga ke dalam OrderProduct
		request.Products[i].Total_price = totalPrice

		// Menambahkan total harga produk dalam pesanan
		total_price += totalPrice
	}

	// Total kembalian
	total_return := request.TotalPaid - total_price
	TotalPaid := request.TotalPaid

	// insert data to orders
	firstLetter := strings.ToUpper(string(username[0]))
	rand.Seed(time.Now().UnixNano())
	randomDigits := rand.Intn(900) + 100
	receipt_code := fmt.Sprintf("%s%d", firstLetter, randomDigits) //generate random string for receipt code

	currentTime := time.Now() //waktu saat ini
	orderResult, err := config.DB.Exec("INSERT INTO orders (user_id, name, payment_id, total_price, total_paid, total_return, receipt_code, created_at,  updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, username, paymentID, total_price, TotalPaid, total_return, receipt_code, currentTime, currentTime)
	if err != nil {
		errorMessage := fmt.Sprintf("Gagal menyimpan orders ke database: %v", err)

		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}
	lastInsertID, err := orderResult.LastInsertId()
	if err != nil {
		responses.ErrorResponse(w, "Gagal mendapatkan ID orders yang baru", http.StatusInternalServerError)
		return
	}
	// insert data to OrderProducts
	var productsInfo []OrderProduct
	for _, orderProduct := range request.Products {
		Qty := orderProduct.Qty
		productID := orderProduct.ProductID
		TotalPrice := orderProduct.Total_price

		// Insert data ke dalam order_products
		orderProductsResult, err := config.DB.Exec("INSERT INTO order_products (order_id, product_id, qty, total_price, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			lastInsertID, productID, Qty, TotalPrice, currentTime, currentTime)
		if err != nil {
			errorMessage := fmt.Sprintf("Gagal menyimpan order_products ke database: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}

		// Dapatkan ID dari order_products yang baru saja di-insert
		lastOrderId, err := orderProductsResult.LastInsertId()
		if err != nil {
			responses.ErrorResponse(w, "Gagal mendapatkan ID orders_products yang baru", http.StatusInternalServerError)
			return
		}

		// Simpan data ke dalam slice productsInfo
		var product OrderProduct
		product.Id = lastOrderId
		product.Order_id = lastInsertID
		product.ProductID = productID
		productsInfo = append(productsInfo, product)
	}

	type Response struct {
		ID            int64          `json:"id"`
		UserID        int            `json:"user_id"`
		PaymentTypeID int            `json:"payment_type_id"`
		TotalPrice    int            `json:"total_price"`
		TotalPaid     int            `json:"total_paid"`
		TotalReturn   int            `json:"total_return"`
		ReceiptID     string         `json:"receipt_id"`
		Products      []OrderProduct `json:"products"`
		PaymentType   Payments       `json:"payment_type"`
		UpdatedAt     string         `json:"updated_at"`
		CreatedAt     string         `json:"created_at"`
	}
	responseData := Response{
		ID:            lastInsertID,
		UserID:        userID,
		PaymentTypeID: paymentID,
		TotalPrice:    total_price,
		TotalPaid:     request.TotalPaid,
		TotalReturn:   total_return,
		ReceiptID:     receipt_code,
		Products:      productsInfo,
		PaymentType:   payment,
		UpdatedAt:     currentTime.Format(time.RFC3339),
		CreatedAt:     currentTime.Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "success", responseData, http.StatusCreated)
}

func ListOrders(w http.ResponseWriter, r *http.Request) {
	type OrderProduct struct {
		Id          *int64 `json:"id"`
		Order_id    *int64 `json:"order_id"`
		ProductID   *int64 `json:"product_id"`
		Qty         *int64 `json:"qty"`
		Total_price *int64 `json:"total_normal_price"`
	}

	type Products struct {
		ID        *int64  `json:"id"`
		SKU       *string `json:"sku"`
		Name      *string `json:"name"`
		Stock     *string `json:"stock"`
		Price     *string `json:"price"`
		Image     *string `json:"image"`
		CreatedAt *string `json:"created_at"`
		UpdatedAt *string `json:"updated_at"`
	}
	type Payments struct {
		Id   int    `json:"payment_id"`
		Name string `json:"name"`
		Type string `json:"type"`
		Logo string `json:"logo"`
	}
	type Orders struct {
		ID              int    `json:"id"`
		User_id         int    `json:"user_id"`
		Payment_type_id int    `json:"payment_type_id"`
		Total_price     int    `json:"total_price"`
		Total_paid      int    `json:"total_paid"`
		Total_return    int    `json:"total_return"`
		Receipt_id      string `json:"receipt_id"`
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
	// get data orders from db
	query := "SELECT id, user_id, payment_id, total_price, total_paid, total_return, receipt_code FROM orders WHERE 1=1"
	args := []interface{}{}

	if categoryIDStr != "" {
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			responses.ErrorResponse(w, "Invalid 'categoryId' parameter", http.StatusBadRequest)
			return
		}
		query += " AND (category_id = ? OR category_id IS NULL)"
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
		errorMessage := fmt.Sprintf("Gagal ambil data  orders: %v", err)

		responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Orders
	var orderProducts []OrderProduct
	var products []Products
	var payments Payments
	var product Products

	for rows.Next() {
		var order Orders
		err := rows.Scan(&order.ID, &order.User_id, &order.Payment_type_id, &order.Total_price, &order.Total_paid, &order.Total_return, &order.Receipt_id)
		if err != nil {
			errorMessage := fmt.Sprintf("Gagal masukkan data orders ke var: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)

		// end orders
		defer rows.Close()
		// get data payment from id payment in orders
		err = config.DB.QueryRow("SELECT id, name, type, logo FROM payments WHERE id=?", order.Payment_type_id).Scan(&payments.Id, &payments.Name, &payments.Type, &payments.Logo)
		if err != nil {
			if err == sql.ErrNoRows {
				errorMessage := fmt.Sprintf("Gagal ambil data payments: %v", err)

				responses.ErrorResponse(w, errorMessage, http.StatusNotFound)
				return
			}
			responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// end payments

		for rows.Next() {
			// get data OrderProducts
			orderId := order.ID
			fmt.Print(orderId)

			query := "SELECT id, order_id, product_id, qty, total_price FROM order_products WHERE order_id = ?"
			rows, err := config.DB.Query(query, orderId)
			if err != nil {
				errorMessage := fmt.Sprintf("Gagal ambil data order_products: %v", err)
				responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
				return
			}
			defer rows.Close() // Tutup rows setelah selesai menggunakan

			// Loop rows dan memasukkan data ke dalam slice orderProducts
			for rows.Next() {
				var op OrderProduct
				err := rows.Scan(&op.Id, &op.Order_id, &op.ProductID, &op.Qty, &op.Total_price)
				if err != nil {
					errorMessage := fmt.Sprintf("Gagal masukkan data order_products: %v", err)
					responses.ErrorResponse(w, errorMessage, http.StatusInternalServerError)
					return
				}
				orderProducts = append(orderProducts, op)
			}
			// end order_products
			for _, orderProduct := range orderProducts {
				// Pastikan orderProduct.ProductID tidak nil sebelum mengakses nilainya
				if orderProduct.ProductID != nil {
					productID := *orderProduct.ProductID
					fmt.Println("Product ID:", productID)
				} else {
					fmt.Println("Product ID is nil")
				}
			}
			// end order-product
			// get data products from id products in orders
			for _, orderProduct := range orderProducts {
				// Pastikan productID tidak nil
				if orderProduct.ProductID == nil {
					errorMessage := "Product ID is nil"
					responses.ErrorResponse(w, errorMessage, http.StatusBadRequest)
					return
				}
				// Query produk berdasarkan productID
				err := config.DB.QueryRow("SELECT id, sku, name, stock, price, image, created_at, updated_at FROM products WHERE id=?", *orderProduct.ProductID).Scan(
					&product.ID, &product.SKU, &product.Name, &product.Stock, &product.Price, &product.Image, &product.CreatedAt, &product.UpdatedAt,
				)

				if err != nil {
					if err == sql.ErrNoRows {
						errorMessage := fmt.Sprintf("Produk dengan ID %d tidak ditemukan", *orderProduct.ProductID)
						responses.ErrorResponse(w, errorMessage, http.StatusNotFound)
						return
					}
					// Handle other database errors
					responses.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Tambahkan produk ke slice products
				products = append(products, product)

				log.Println("Mengecek orderProduct:", *orderProduct.ProductID)
				log.Printf("Data produk: %+v\n", product)
			}
			// end products
		}

	}
	total := len(orders)

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"meta": map[string]interface{}{
				"total": total,
				"limit": limit,
				"skip":  skip,
			},
			"orders":   orders,
			"products": products,
			"payments": payments,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	responses.SuccessResponse(w, "Success", response, http.StatusOK)
}
