package migration

import (
	"database/sql"
	"log"
)

// OrderMigration digunakan untuk menjalankan migrasi tabel.
func OrderMigration(db *sql.DB) {
	// SQL statement untuk memeriksa apakah tabel users sudah ada
	checkTableSQL := `
        SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'orders'
    `

	// Menjalankan perintah SQL untuk memeriksa apakah tabel sudah ada
	var tableCount int
	err := db.QueryRow(checkTableSQL).Scan(&tableCount)
	if err != nil {
		// Menangani kesalahan jika terjadi kesalahan saat memeriksa tabel
		log.Fatal(err)
		return
	}

	if tableCount > 0 {
		// Jika tabel sudah ada, tampilkan pesan
		log.Println("Tabel sudah di migrasi")
		return
	}
	// SQL statement untuk membuat tabel orders
	createTableSQL := `
        CREATE TABLE IF NOT EXISTS orders (
            id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT,
            payment_id INT,
            name VARCHAR(255) NOT NULL,
            total_price DECIMAL(10, 2) NOT NULL,
            total_paid DECIMAL(10, 2) NOT NULL,
            total_return DECIMAL(10, 2) NOT NULL,
            receipt_code VARCHAR(255) NOT NULL,
            created_at TIMESTAMP NOT NULL,
            updated_at TIMESTAMP NOT NULL,
            FOREIGN KEY (user_id) REFERENCES users(id),
            FOREIGN KEY (payment_id) REFERENCES payments(id)
        )
    `

	// Menjalankan perintah SQL untuk membuat tabel
	_, err = db.Exec(createTableSQL)
	if err != nil {
		// Menangani kesalahan jika terjadi kesalahan saat migrasi
		log.Fatal(err)
		return
	}

	// Pesan sukses jika migrasi berhasil
	log.Println("Migrasi tabel berhasil")
}
