package migration

import (
	"database/sql"
	"log"
)

// Migrate digunakan untuk menjalankan migrasi tabel.
func UserMigrate(db *sql.DB) {
	// SQL statement untuk memeriksa apakah tabel users sudah ada
	checkTableSQL := `
        SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'users'
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

	// SQL statement untuk membuat tabel users
	createTableSQL := `
        CREATE TABLE IF NOT EXISTS users (
            id INT AUTO_INCREMENT PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            password VARCHAR(255) NOT NULL,
            email VARCHAR(255) NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL,
            updated_at TIMESTAMP NOT NULL
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
	log.Println("Migrasi tabel users berhasil")
}
