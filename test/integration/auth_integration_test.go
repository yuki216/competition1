package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain/entity"
	"github.com/fixora/fixora/infrastructure/adapter/postgres"
	"github.com/fixora/fixora/infrastructure/config"
	"github.com/fixora/fixora/infrastructure/http/handler"
	"github.com/fixora/fixora/infrastructure/http/middleware"
	"github.com/fixora/fixora/infrastructure/http/response"
	"github.com/fixora/fixora/infrastructure/persistence/usecase"
	"github.com/fixora/fixora/infrastructure/service/jwt"
	"github.com/fixora/fixora/infrastructure/service/logger"
	"github.com/fixora/fixora/infrastructure/service/password"
	"github.com/fixora/fixora/infrastructure/service/ratelimit"
	"github.com/fixora/fixora/infrastructure/service/recaptcha"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

type AuthIntegrationTestSuite struct {
	db          *sql.DB
	authHandler *handler.AuthHandler
	authUseCase inbound.AuthUseCase
	config      *config.Config
	ctx         context.Context
}

func setupAuthIntegrationTest(t *testing.T) *AuthIntegrationTestSuite {
	ctx := context.Background()

	// Load config
	cfg := &config.Config{
		DatabaseURL:      "postgres://postgres:postgres@localhost:5432/auth_service_test?sslmode=disable",
		JWTSecret:        "test-secret-key",
		JWTAlgorithm:     "HS256",
		AccessTokenTTL:   15 * 60,           // 15 minutes in seconds
		RefreshTokenTTL:  30 * 24 * 60 * 60, // 30 days in seconds
		RefreshTokenSalt: "test-salt",
	}

	// Ensure test database exists and has schema
	db, err := ensureTestDatabaseAndSchema(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to prepare test database: %v", err)
	}

	// Clean up database
	if err := cleanupDatabase(db); err != nil {
		t.Fatalf("Failed to cleanup database: %v", err)
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepositoryAdapter(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepositoryAdapter(db, cfg.RefreshTokenSalt)

	// Initialize services
	tokenService, err := jwt.NewJWTService(cfg)
	if err != nil {
		t.Fatalf("Failed to create token service: %v", err)
	}
	passwordService := password.NewBcryptPasswordService(10)
	// Initialize recaptcha and rate limit services for tests (noop)
	recaptchaService := recaptcha.NewNoopRecaptchaService(nil)
	rateLimitService, _ := ratelimit.NewRateLimitService(ratelimit.RateLimitConfig{Enabled: false}, logrus.New())
	structuredLogger := logger.NewStructuredLogger(logger.LoggerConfig{Level: "INFO", Format: "json", ServiceName: "test"})

	// Initialize use case
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		refreshTokenRepo,
		tokenService,
		passwordService,
		recaptchaService,
		rateLimitService,
		structuredLogger,
		time.Duration(cfg.AccessTokenTTL)*time.Second,
		time.Duration(cfg.RefreshTokenTTL)*time.Second,
	)

	// Initialize handler
	authHandler := handler.NewAuthHandler(authUseCase)

	return &AuthIntegrationTestSuite{
		db:          db,
		authHandler: authHandler,
		authUseCase: authUseCase,
		config:      cfg,
		ctx:         ctx,
	}
}

func cleanupDatabase(db *sql.DB) error {
	queries := []string{
		"TRUNCATE TABLE refresh_tokens CASCADE",
		"TRUNCATE TABLE users CASCADE",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthIntegrationTestSuite) cleanup(t *testing.T) {
	if err := cleanupDatabase(s.db); err != nil {
		t.Logf("Failed to cleanup database: %v", err)
	}
	if err := s.db.Close(); err != nil {
		t.Logf("Failed to close database connection: %v", err)
	}
}

func TestLoginIntegration(t *testing.T) {
	suite := setupAuthIntegrationTest(t)
	defer suite.cleanup(t)

	tests := []struct {
		name            string
		setup           func()
		requestBody     interface{}
		expectedStatus  int
		expectedSuccess bool
		expectedMessage string
		checkResponse   func(t *testing.T, result response.Envelope)
	}{
		{
			name: "successful login",
			setup: func() {
				// Create test user with bcrypt hashed password matching "password123"
				ps := password.NewBcryptPasswordService(10)
				hash, err := ps.HashPassword("password123")
				if err != nil {
					t.Fatal(err)
				}
				user := entity.NewUser("test-user-id", "Test User", "test@example.com", hash, "user", "active")
				if _, err := suite.db.ExecContext(suite.ctx, `
					INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`, user.ID, user.Name, user.Email, user.Password, user.Role, user.Status, user.CreatedAt, user.UpdatedAt); err != nil {
					t.Fatal(err)
				}
				// Verify the stored hash matches what we generated
				var storedHash string
				if err := suite.db.QueryRowContext(suite.ctx, `SELECT password FROM users WHERE email = $1`, user.Email).Scan(&storedHash); err != nil {
					t.Fatal(err)
				}
				if storedHash == "" || storedHash == "password123" {
					t.Fatalf("stored password invalid: %q", storedHash)
				}
			},
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
			expectedMessage: "success",
			checkResponse: func(t *testing.T, result response.Envelope) {
				if result.Status != true {
					t.Error("Expected status to be true")
				}
				data, ok := result.Data.(map[string]interface{})
				if !ok {
					t.Fatal("Expected data to be an object")
				}
				if _, ok := data["access_token"]; !ok {
					t.Error("Expected access_token in response")
				}
				if _, ok := data["expires_in"]; !ok {
					t.Error("Expected expires_in in response")
				}
			},
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password123",
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedSuccess: false,
			expectedMessage: "Invalid email format",
		},
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedSuccess: false,
			expectedMessage: "Invalid email format",
		},
		{
			name: "missing password",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedStatus:  http.StatusUnprocessableEntity,
			expectedSuccess: false,
			expectedMessage: "Password is required",
		},
		{
			name: "invalid credentials",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "wrong-password",
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Email atau password salah",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Correlation-ID", "test-correlation-id")

			w := httptest.NewRecorder()
			suite.authHandler.Login(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var result response.Envelope
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}

			if result.Status != tt.expectedSuccess {
				t.Errorf("Expected status %v, got %v", tt.expectedSuccess, result.Status)
			}

			if tt.expectedMessage != "" && result.Message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, result.Message)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, result)
			}
		})
	}
}

func TestRefreshIntegration(t *testing.T) {
	suite := setupAuthIntegrationTest(t)
	defer suite.cleanup(t)

	// Setup test data
	ps := password.NewBcryptPasswordService(10)
	hash, err := ps.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	user := entity.NewUser("test-user-id", "Test User", "test@example.com", hash, "user", "active")
	if _, err := suite.db.ExecContext(suite.ctx, `
		INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.Name, user.Email, user.Password, user.Role, user.Status, user.CreatedAt, user.UpdatedAt); err != nil {
		t.Fatal(err)
	}

	// Create refresh token via repository (hash stored as token_hash)
	refreshToken := entity.NewRefreshToken("test-refresh-id", user.ID, "test-refresh-token", time.Now().Add(30*24*time.Hour))
	refreshRepo := postgres.NewRefreshTokenRepositoryAdapter(suite.db, suite.config.RefreshTokenSalt)
	if err := refreshRepo.Create(suite.ctx, refreshToken); err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	tests := []struct {
		name            string
		refreshToken    string
		expectedStatus  int
		expectedSuccess bool
		expectedMessage string
	}{
		{
			name:            "successful refresh",
			refreshToken:    "test-refresh-token",
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
			expectedMessage: "success",
		},
		{
			name:            "invalid refresh token",
			refreshToken:    "invalid-token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Invalid or expired refresh token",
		},
		{
			name:            "missing refresh token",
			refreshToken:    "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Refresh token required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/refresh", nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")

			if tt.refreshToken != "" {
				req.Header.Set("Refresh-Token", tt.refreshToken)
			}

			w := httptest.NewRecorder()
			suite.authHandler.Refresh(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var result response.Envelope
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}

			if result.Status != tt.expectedSuccess {
				t.Errorf("Expected status %v, got %v", tt.expectedSuccess, result.Status)
			}

			if tt.expectedMessage != "" && result.Message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, result.Message)
			}
		})
	}
}

func TestLogoutIntegration(t *testing.T) {
	suite := setupAuthIntegrationTest(t)
	defer suite.cleanup(t)

	// Setup test data
	ps := password.NewBcryptPasswordService(10)
	hash, err := ps.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	user := entity.NewUser("test-user-id", "Test User", "test@example.com", hash, "user", "active")
	if _, err := suite.db.ExecContext(suite.ctx, `
		INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.Name, user.Email, user.Password, user.Role, user.Status, user.CreatedAt, user.UpdatedAt); err != nil {
		t.Fatal(err)
	}

	// Create refresh token via repository (hash stored as token_hash)
	refreshToken := entity.NewRefreshToken("test-refresh-id", user.ID, "test-refresh-token", time.Now().Add(30*24*time.Hour))
	refreshRepo := postgres.NewRefreshTokenRepositoryAdapter(suite.db, suite.config.RefreshTokenSalt)
	if err := refreshRepo.Create(suite.ctx, refreshToken); err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Generate access token for logout tests using same service as authUseCase
	// We need to get the token service from authUseCase somehow. For now, let's create a new one but with consistent config
	tokenService, err := jwt.NewJWTService(suite.config)
	if err != nil {
		t.Fatal(err)
	}
	accessToken, err := tokenService.GenerateAccessToken(outbound.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		authHeader      string
		expectedStatus  int
		expectedSuccess bool
		expectedMessage string
	}{
		{
			name:            "successful logout",
			authHeader:      "Bearer " + accessToken,
			expectedStatus:  http.StatusNoContent,
			expectedSuccess: true,
			expectedMessage: "",
		},
		{
			name:            "missing auth header",
			authHeader:      "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Authorization header required",
		},
		{
			name:            "invalid auth header format",
			authHeader:      "InvalidFormat token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Invalid authorization header format",
		},
		{
			name:            "invalid token",
			authHeader:      "Bearer invalid-token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Invalid or expired token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/logout", nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Wrap with auth middleware to enforce Authorization checks
			authMw := middleware.NewAuthMiddleware(tokenService)
			h := authMw.RequireAuth(suite.authHandler.Logout)

			w := httptest.NewRecorder()
			h(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Only parse body when not 204 No Content
			if tt.expectedStatus != http.StatusNoContent {
				var result response.Envelope
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}

				if result.Status != tt.expectedSuccess {
					t.Errorf("Expected status %v, got %v", tt.expectedSuccess, result.Status)
				}

				if tt.expectedMessage != "" && result.Message != tt.expectedMessage {
					t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, result.Message)
				}
			}
		})
	}
}

func TestMeIntegration(t *testing.T) {
	suite := setupAuthIntegrationTest(t)
	defer suite.cleanup(t)

	// Setup test data
	user := entity.NewUser("test-user-id", "Test User", "test@example.com", "hashed-password", "user", "active")
	if _, err := suite.db.ExecContext(suite.ctx, `
		INSERT INTO users (id, name, email, password, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.Name, user.Email, user.Password, user.Role, user.Status, user.CreatedAt, user.UpdatedAt); err != nil {
		t.Fatal(err)
	}

	// Generate access token using same config as authUseCase
	tokenService, err := jwt.NewJWTService(suite.config)
	if err != nil {
		t.Fatal(err)
	}
	accessToken, err := tokenService.GenerateAccessToken(outbound.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		authHeader      string
		expectedStatus  int
		expectedSuccess bool
		expectedMessage string
		checkResponse   func(t *testing.T, result response.Envelope)
	}{
		{
			name:            "successful me request",
			authHeader:      "Bearer " + accessToken,
			expectedStatus:  http.StatusOK,
			expectedSuccess: true,
			expectedMessage: "success",
			checkResponse: func(t *testing.T, result response.Envelope) {
				data, ok := result.Data.(map[string]interface{})
				if !ok {
					t.Fatal("Expected data to be an object")
				}

				if data["id"] != "test-user-id" {
					t.Errorf("Expected id 'test-user-id', got '%v'", data["id"])
				}

				if data["email"] != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got '%v'", data["email"])
				}
			},
		},
		{
			name:            "missing auth header",
			authHeader:      "",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Authorization header required",
		},
		{
			name:            "invalid auth header format",
			authHeader:      "InvalidFormat token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Invalid authorization header format",
		},
		{
			name:            "invalid token",
			authHeader:      "Bearer invalid-token",
			expectedStatus:  http.StatusUnauthorized,
			expectedSuccess: false,
			expectedMessage: "Invalid or expired token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/auth/me", nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Wrap with auth middleware to enforce Authorization checks
			ts, err := jwt.NewJWTService(suite.config)
			if err != nil {
				t.Fatal(err)
			}
			authMw := middleware.NewAuthMiddleware(ts)
			h := authMw.RequireAuth(suite.authHandler.Me)

			w := httptest.NewRecorder()
			h(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var result response.Envelope
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}

			if result.Status != tt.expectedSuccess {
				t.Errorf("Expected status %v, got %v", tt.expectedSuccess, result.Status)
			}

			if tt.expectedMessage != "" && result.Message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, result.Message)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, result)
			}
		})
	}
}

// ensureTestDatabaseAndSchema makes sure the target DB exists and required tables are created
func ensureTestDatabaseAndSchema(dsn string) (*sql.DB, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid database url: %w", err)
	}
	// Extract db name
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return nil, fmt.Errorf("database name missing in DSN")
	}
	// Connect to admin 'postgres' database to create db if missing
	adminURL := *u
	adminURL.Path = "/postgres"
	adminDB, err := sql.Open("postgres", adminURL.String())
	if err != nil {
		return nil, fmt.Errorf("connect admin db: %w", err)
	}
	defer adminDB.Close()
	// Check existence
	var dummy int
	q := `SELECT 1 FROM pg_database WHERE datname = $1`
	if err := adminDB.QueryRow(q, dbName).Scan(&dummy); err != nil {
		// Not found; create database
		createStmt := fmt.Sprintf("CREATE DATABASE \"%s\"", strings.ReplaceAll(dbName, "\"", "\"\""))
		if _, err2 := adminDB.Exec(createStmt); err2 != nil {
			return nil, fmt.Errorf("create database: %w", err2)
		}
	}
	// Connect to target DB
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect target db: %w", err)
	}
	// Create schema tables if not exist
	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	// Drop existing table if it exists (to handle schema changes)
	db.Exec(`DROP TABLE IF EXISTS users CASCADE;`)
	db.Exec(`DROP TABLE IF EXISTS refresh_tokens CASCADE;`)

	// users table with updated schema
	users := `
	CREATE TABLE users (
	    id VARCHAR(255) PRIMARY KEY,
	    name VARCHAR(255) NOT NULL DEFAULT '',
	    email VARCHAR(255) UNIQUE NOT NULL,
	    password VARCHAR(255) NOT NULL,
	    role VARCHAR(20) NOT NULL DEFAULT 'employee',
	    status VARCHAR(20) NOT NULL DEFAULT 'active',
	    deleted_at TIMESTAMP NULL,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    CONSTRAINT chk_user_role CHECK (role IN ('admin', 'user', 'superadmin', 'employee')),
	    CONSTRAINT chk_user_status CHECK (status IN ('active', 'inactive'))
	);`
	if _, err := db.Exec(users); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX idx_users_email ON users(email);`,
		`CREATE INDEX idx_users_role ON users(role);`,
		`CREATE INDEX idx_users_status ON users(status);`,
		`CREATE INDEX idx_users_deleted_at ON users(deleted_at);`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("create users index: %w", err)
		}
	}
	// refresh_tokens table with token_hash and revoked
	refresh := `
	CREATE TABLE refresh_tokens (
	    id VARCHAR(255) PRIMARY KEY,
	    user_id VARCHAR(255) NOT NULL,
	    token_hash BYTEA UNIQUE NOT NULL,
	    expires_at TIMESTAMP NOT NULL,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    revoked BOOLEAN NOT NULL DEFAULT FALSE,
	    revoked_at TIMESTAMP NULL,
	    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(refresh); err != nil {
		return fmt.Errorf("create refresh_tokens table: %w", err)
	}

	refreshIndexes := []string{
		`CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);`,
		`CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);`,
		`CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);`,
		`CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(revoked);`,
	}

	for _, idx := range refreshIndexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("create refresh_tokens index: %w", err)
		}
	}

	return nil
}
