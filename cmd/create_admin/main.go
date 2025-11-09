package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/google/uuid"

	"github.com/fixora/fixora/domain/entity"
	"github.com/fixora/fixora/infrastructure/adapter/postgres"
	"github.com/fixora/fixora/infrastructure/config"
	"github.com/fixora/fixora/infrastructure/service/password"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize repository
	userRepo := postgres.NewUserRepositoryAdapter(db)

	// Get admin credentials from command line args or use defaults
	email := "admin@vibe.com"
	userPassword := "admin123"
	name := "Administrator"
	role := "admin"

	if len(os.Args) > 1 {
		email = os.Args[1]
	}
	if len(os.Args) > 2 {
		userPassword = os.Args[2]
	}
	if len(os.Args) > 3 {
		name = os.Args[3]
	}

	// Hash password
	passwordService := password.NewBcryptPasswordService(10)
	hashedPassword, err := passwordService.HashPassword(userPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create admin user
	adminUser := entity.NewUserWithDefaults(
		uuid.New().String(),
		name,
		email,
		hashedPassword,
		role,
	)

	// Save user to database
	err = userRepo.Create(ctx, adminUser)
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("✅ Admin user created successfully!\n")
	fmt.Printf("📧 Email: %s\n", email)
	fmt.Printf("👤 Name: %s\n", name)
	fmt.Printf("🔑 Password: %s\n", userPassword)
	fmt.Printf("🎭 Role: %s\n", role)
	fmt.Printf("🆔 ID: %s\n", adminUser.ID)
}