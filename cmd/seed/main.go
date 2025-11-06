package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	email := getenvDefault("SEED_USER_EMAIL", "demo@example.com")
	password := getenvDefault("SEED_USER_PASSWORD", "Demo1234!")
	role := getenvDefault("SEED_USER_ROLE", "employee")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	// hash password with bcrypt cost 10
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// upsert user by email; let DB generate UUID id by default
	query := `
	INSERT INTO users (email, password, role, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (email) DO UPDATE SET
	  password = EXCLUDED.password,
	  role = EXCLUDED.role,
	  updated_at = EXCLUDED.updated_at
	RETURNING id
	`

	now := time.Now()
	var id string
	err = db.QueryRow(query, email, string(hash), role, now, now).Scan(&id)
	if err != nil {
		log.Fatalf("failed to seed user: %v", err)
	}

	fmt.Printf("Seeded user: email=%s password=%s role=%s id=%s\n", email, password, role, id)
}

func getenvDefault(k, d string) string {
	v := os.Getenv(k)
	if v == "" { return d }
	return v
}