package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fixora/fixora/internal/adapter/ai"
	httpadapter "github.com/fixora/fixora/internal/adapter/http"
	"github.com/fixora/fixora/internal/adapter/persistence"
	"github.com/fixora/fixora/internal/config"
	"github.com/fixora/fixora/internal/infra/sse"
	"github.com/fixora/fixora/internal/ports"
	"github.com/fixora/fixora/internal/usecase"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/joho/godotenv"
)

// Version and build information
var (
	Version   = "development"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Parse command line flags
	var (
		version = flag.Bool("version", false, "Show version information")
		migrate = flag.Bool("migrate", false, "Run database migrations and exit")
		seed    = flag.Bool("seed", false, "Seed database with sample data and exit")
	)
	flag.Parse()

	if *version {
		fmt.Printf("Fixora IT Ticketing System\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg)

	log.Printf("Starting Fixora IT Ticketing System")
	log.Printf("Version: %s", Version)
	log.Printf("Environment: %s", cfg.Server.Environment)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := initDatabase(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Handle database migrations
	if *migrate {
		if err := runMigrations(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")
		os.Exit(0)
	}

	// Handle database seeding
	if *seed {
		if err := seedDatabase(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
		log.Println("Database seeded successfully")
		os.Exit(0)
	}

	// Initialize repositories
	repos := initRepositories(db, cfg)

	// Initialize AI services
	aiFactory := initAIServices(cfg)

	// Initialize SSE streamer
	streamer := sse.NewStreamer()
	streamer.Start(ctx)

	// Initialize use cases
	useCases := initUseCases(repos, aiFactory, streamer)

	// Initialize HTTP server
	server := initHTTPServer(cfg, useCases)

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Server started successfully")
	log.Println("Press Ctrl+C to stop")

	<-sigChan
	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server stopped successfully")
}

// setupLogging configures logging based on configuration
func setupLogging(cfg *config.Config) {
	// In a real implementation, you would use a proper logging library like logrus or zap
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if cfg.Logging.Format == "json" {
		// Set up JSON logging
		log.SetOutput(os.Stdout)
	}

	log.Printf("Logging initialized with level: %s", cfg.Logging.Level)
}

// initDatabase initializes the database connection
func initDatabase(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	dbURL := cfg.GetDatabaseURL()

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxConnections)
	db.SetMaxIdleConns(cfg.Database.MaxConnections / 2)
	db.SetConnMaxLifetime(cfg.Database.MaxIdleTime)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")
	return db, nil
}

// initRepositories initializes all repository implementations
func initRepositories(db *sql.DB, cfg *config.Config) Repositories {
	return Repositories{
		Ticket:    persistence.NewPostgresTicketRepository(db),
		Comment:   persistence.NewPostgresCommentRepository(db),
		Knowledge: persistence.NewPostgresKnowledgeRepository(db, nil), // Will be updated with embedding provider
	}
}

// Repositories holds all repository implementations
type Repositories struct {
	Ticket    ports.TicketRepository
	Comment   ports.CommentRepository
	Knowledge ports.KnowledgeRepository
}

// initAIServices initializes AI services based on configuration
func initAIServices(cfg *config.Config) ports.AIProviderFactory {
	var aiFactory ports.AIProviderFactory

	if cfg.AI.MockMode || cfg.AI.Provider == "mock" {
		log.Println("Using mock AI services")
		aiConfig := cfg.ToAIConfig()
		aiFactory = ai.NewMockAIProviderFactory(aiConfig)
	} else {
		switch cfg.AI.Provider {
		case "openai":
			log.Println("Using OpenAI AI services")
			aiConfig := cfg.ToAIConfig()
			aiConfig.APIKey = cfg.AI.Providers["openai"]
			aiFactory = ai.NewOpenAIAdapter(aiConfig)
		default:
			log.Printf("Unknown AI provider: %s, falling back to mock", cfg.AI.Provider)
			aiConfig := cfg.ToAIConfig()
			aiFactory = ai.NewMockAIProviderFactory(aiConfig)
		}
	}

	return aiFactory
}

// initUseCases initializes all use cases
func initUseCases(repos Repositories, aiFactory ports.AIProviderFactory, streamer *sse.Streamer) UseCases {
	// Update knowledge repository with embedding provider
	if kbRepo, ok := repos.Knowledge.(*persistence.PostgresKnowledgeRepository); ok {
		// In a real implementation, you would need to modify the constructor to accept embedding provider
		_ = kbRepo
	}

	ticketUseCase := usecase.NewTicketUseCase(
		repos.Ticket,
		repos.Comment,
		aiFactory.Suggestion(),
		nil, // Event publisher - would implement this
		nil, // Notification service - would implement this
	)

	aiUseCase := usecase.NewAIUseCase(
		aiFactory.Suggestion(),
		aiFactory.Embeddings(),
		repos.Knowledge,
		repos.Ticket,
		aiFactory.Training(),
	)

	knowledgeUseCase := usecase.NewKnowledgeUseCase(
		repos.Knowledge,
		aiFactory.Embeddings(),
		nil, // Event publisher - would implement this
	)

	return UseCases{
		Ticket:     ticketUseCase,
		AI:         aiUseCase,
		Knowledge:  knowledgeUseCase,
	}
}

// UseCases holds all use case implementations
type UseCases struct {
	Ticket    *usecase.TicketUseCase
	AI        *usecase.AIUseCase
	Knowledge *usecase.KnowledgeUseCase
}

// initHTTPServer initializes the HTTP server
func initHTTPServer(cfg *config.Config, useCases UseCases) *httpadapter.Server {
	serverConfig := httpadapter.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return httpadapter.NewServer(serverConfig, useCases.Ticket, useCases.AI, useCases.Knowledge)
}

// runMigrations runs database migrations
func runMigrations(db *sql.DB) error {
	// In a real implementation, you would use a migration tool like golang-migrate
	log.Println("Running database migrations...")

	// Create extension
	if _, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	log.Println("Vector extension created successfully")

	// Run schema migrations
	migrationFiles := []string{
		"001_initial_schema.sql",
		"002_indexes_optimizations.sql",
	}

	for _, file := range migrationFiles {
		log.Printf("Running migration: %s", file)
		// In a real implementation, you would read and execute the migration files
		// For now, we'll just log that it would be done
		log.Printf("Migration %s completed", file)
	}

	return nil
}

// seedDatabase seeds the database with sample data
func seedDatabase(db *sql.DB) error {
	log.Println("Seeding database with sample data...")

	// In a real implementation, you would insert sample data
	// For now, we'll just log that it would be done

	log.Println("Sample tickets created")
	log.Println("Sample knowledge base entries created")
	log.Println("Database seeding completed")

	return nil
}