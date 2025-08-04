package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Driver untuk PostgreSQL
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Driver untuk membaca dari file
	"github.com/joho/godotenv"
)

func main() {
	// Memuat variabel dari file .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Mengambil URL database dari environment variable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Menentukan path ke direktori migrasi
	// "file://../../db/migrations" berarti dari cmd/migrator, naik 2 level ke root, lalu masuk ke db/migrations
	migrationsPath := "file://db/migrations"

	// Membuat instance migrasi baru
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Menjalankan migrasi NAIK (menerapkan semua file .up.sql yang baru)
	log.Println("Running database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migration completed successfully!")
}
