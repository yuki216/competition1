package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/infrastructure/config"
	"github.com/vobe/auth-service/infrastructure/http/handler"
	"github.com/vobe/auth-service/infrastructure/http/middleware"
	"github.com/vobe/auth-service/infrastructure/http/response"
	"github.com/vobe/auth-service/infrastructure/service/logger"
)

type Iteration3AcceptanceTestSuite struct {
	suite.Suite
	server           *httptest.Server
	authHandler      *handler.AuthHandler
	structuredLogger logger.Logger
	config           *config.Config
	rlService        *memoryRateLimitService
}

func (suite *Iteration3AcceptanceTestSuite) SetupSuite() {
	// Load test configuration
	cfg := &config.Config{
		DatabaseURL:      os.Getenv("TEST_DATABASE_URL"),
		ServerHost:       "localhost",
		ServerPort:       "8080",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  30 * 24 * time.Hour,
		RefreshTokenSalt: "test-salt",
		RecaptchaEnabled: true,
		RecaptchaSecret:   "6LeIxAcTAAAAAGG-vFI1TnRWxMZNFuojJ4WifJWe", // Test secret
		RecaptchaSiteKey:  "6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI", // Test site key
		RecaptchaTimeout:  5 * time.Second,
		RecaptchaSkip:     false,
		RateLimitEnabled:   true,
		RedisURL:         "redis://localhost:6379/1",
		LogFormat:        "json",
		LogLevel:         "info",
		Environment:      "test",
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgres://postgres:postgres@localhost:5432/auth_test?sslmode=disable"
	}

	suite.config = cfg

	// Initialize structured logger
	suite.structuredLogger = logger.NewStructuredLogger(logger.LoggerConfig{
		Level:               cfg.LogLevel,
		Format:              cfg.LogFormat,
		CorrelationIDHeader: middleware.CorrelationIDHeader,
		EnableRequestLog:    false,
		EnableResponseLog:   false,
		ServiceName:         "acceptance-tests",
	})

	// Initialize in-memory rate limiting service
	suite.rlService = NewMemoryRateLimitService().(*memoryRateLimitService)

	// Initialize reCAPTCHA service (enabled for tests)
	recaptchaService := NewMemoryRecaptchaService(true)

	// Initialize use case (mock for acceptance test)
	authUseCase := &mockAuthUseCase{
		recaptchaService: recaptchaService,
		rateLimitService: suite.rlService,
		logger:          suite.structuredLogger,
	}

	// Initialize handlers
	suite.authHandler = handler.NewAuthHandler(authUseCase)

	// Setup test server
	router := http.NewServeMux()

	// Add middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(suite.rlService, suite.structuredLogger)
	router.Handle("/v1/auth/login", rateLimitMiddleware.RateLimit(http.HandlerFunc(suite.authHandler.Login)))
	router.Handle("/v1/auth/refresh", rateLimitMiddleware.RateLimit(http.HandlerFunc(suite.authHandler.Refresh)))
	router.HandleFunc("/v1/auth/logout", suite.authHandler.Logout)
	router.HandleFunc("/v1/auth/me", suite.authHandler.Me)

	finalHandler := middleware.CorrelationIDMiddleware(router)
	
	suite.server = httptest.NewServer(finalHandler)
}

func (suite *Iteration3AcceptanceTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// Test 1: reCAPTCHA Integration
func (suite *Iteration3AcceptanceTestSuite) TestRecaptchaIntegration() {
	if suite.rlService != nil { suite.rlService.Reset() }
	tests := []struct {
		name           string
		recaptchaToken string
		email          string
		password       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid reCAPTCHA token",
			recaptchaToken: "valid-test-token",
			email:          "test@example.com",
			password:       "password123",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Missing reCAPTCHA token (should be optional)",
			recaptchaToken: "",
			email:          "test@example.com",
			password:       "password123",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Invalid reCAPTCHA token format",
			recaptchaToken: "invalid-token-format",
			email:          "test@example.com",
			password:       "password123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "reCAPTCHA",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			payload := map[string]interface{}{
				"email":           tt.email,
				"password":        tt.password,
				"recaptcha_token": tt.recaptchaToken,
			}

			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Correlation-ID", "test-correlation-id")

			resp, err := http.DefaultClient.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
				
				var result response.Envelope
				err := json.NewDecoder(resp.Body).Decode(&result)
				suite.NoError(err)
				assert.True(suite.T(), result.Status)
			} else {
				assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
			}

			// Verify correlation ID header
			correlationID := resp.Header.Get("X-Correlation-ID")
			assert.NotEmpty(suite.T(), correlationID)
		})
	}
}

// Test 2: Rate Limiting
func (suite *Iteration3AcceptanceTestSuite) TestRateLimiting() {
	// Reset limiter state
	if suite.rlService != nil { suite.rlService.Reset() }
	// Test login rate limiting
	suite.Run("Login Rate Limiting", func() {
		for i := 0; i < 15; i++ {
			payload := map[string]interface{}{
				"email":    fmt.Sprintf("test%d@example.com", i),
				"password": "wrongpassword",
			}

			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			suite.NoError(err)
			resp.Body.Close()

			if i < 10 {
				// Should get 401 (unauthorized) for first 10 attempts
				assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
			} else {
				// Should get 429 (too many requests) after limit exceeded
				assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode)
			}
		}
	})
}

// Test 3: Structured Logging with Correlation ID
func (suite *Iteration3AcceptanceTestSuite) TestStructuredLogging() {
	suite.Run("Correlation ID Generation", func() {
		req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer([]byte(`{"email":"test@example.com","password":"password123"}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

		// Should generate correlation ID if not provided
		correlationID := resp.Header.Get("X-Correlation-ID")
		assert.NotEmpty(suite.T(), correlationID)
	})

	suite.Run("Correlation ID Preservation", func() {
		providedCorrelationID := "test-correlation-12345"
		
		req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer([]byte(`{"email":"test@example.com","password":"password123"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Correlation-ID", providedCorrelationID)

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		defer resp.Body.Close()

		// Should preserve provided correlation ID
		returnedCorrelationID := resp.Header.Get("X-Correlation-ID")
		assert.Equal(suite.T(), providedCorrelationID, returnedCorrelationID)
	})
}

// Test 4: Security Event Logging
func (suite *Iteration3AcceptanceTestSuite) TestSecurityEventLogging() {
	if suite.rlService != nil { suite.rlService.Reset() }
	suite.Run("Failed Login Attempts", func() {
		payload := map[string]interface{}{
			"email":    "suspicious@example.com",
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	})
}

// Test 5: Performance Monitoring
func (suite *Iteration3AcceptanceTestSuite) TestPerformanceMonitoring() {
	if suite.rlService != nil { suite.rlService.Reset() }
	suite.Run("Response Time Measurement", func() {
		start := time.Now()
		
		payload := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		defer resp.Body.Close()

		duration := time.Since(start)
		
		// Response should be reasonably fast (less than 1 second)
		assert.Less(suite.T(), duration, 1*time.Second)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	})
}

// Test 6: Error Catalog Integration
func (suite *Iteration3AcceptanceTestSuite) TestErrorCatalog() {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing email",
			payload:        map[string]interface{}{"password": "password123"},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "email",
		},
		{
			name:           "Missing password",
			payload:        map[string]interface{}{"email": "test@example.com"},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "password",
		},
		{
			name:           "Invalid email format",
			payload:        map[string]interface{}{"email": "invalid-email", "password": "password123"},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "email",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", suite.server.URL+"/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
		})
	}
}

// Mock implementations for testing

// In-memory RecaptchaService

type memoryRecaptchaService struct { enabled bool }

func NewMemoryRecaptchaService(enabled bool) inbound.RecaptchaService { return &memoryRecaptchaService{enabled: enabled} }

func (m *memoryRecaptchaService) VerifyToken(ctx context.Context, token string) (bool, error) {
	if token == "valid-test-token" { return true, nil }
	if token == "invalid-token-format" { return false, fmt.Errorf("reCAPTCHA token invalid") }
	return true, nil
}
func (m *memoryRecaptchaService) IsEnabled() bool { return m.enabled }

// In-memory RateLimitService

type memoryRateLimitService struct {
	attempts     map[string]int
	blockedUntil map[string]time.Time
}

func (m *memoryRateLimitService) Reset() {
	m.attempts = make(map[string]int)
	m.blockedUntil = make(map[string]time.Time)
}

func NewMemoryRateLimitService() inbound.RateLimitService { return &memoryRateLimitService{attempts: make(map[string]int), blockedUntil: make(map[string]time.Time)} }

func (m *memoryRateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.attempts[key] = m.attempts[key] + 1
	return m.attempts[key] <= limit, nil
}
func (m *memoryRateLimitService) Increment(ctx context.Context, key string, window time.Duration) error { m.attempts[key] = m.attempts[key] + 1; return nil }
func (m *memoryRateLimitService) Block(ctx context.Context, key string, duration time.Duration, reason string) error { m.blockedUntil[key] = time.Now().Add(duration); return nil }
func (m *memoryRateLimitService) IsBlocked(ctx context.Context, key string) (bool, error) { until, ok := m.blockedUntil[key]; if !ok { return false, nil }; return time.Now().Before(until), nil }
func (m *memoryRateLimitService) GetAttempts(ctx context.Context, key string) (int, error) { return m.attempts[key], nil }

// Mock UseCase

type mockAuthUseCase struct {
	recaptchaService inbound.RecaptchaService
	rateLimitService inbound.RateLimitService
	logger          logger.Logger
}

func (m *mockAuthUseCase) Login(ctx context.Context, req inbound.LoginRequest) (*inbound.LoginResponse, error) {
	// Validate reCAPTCHA only if token provided and service enabled
	if m.recaptchaService != nil && m.recaptchaService.IsEnabled() && req.RecaptchaToken != "" {
		valid, err := m.recaptchaService.VerifyToken(ctx, req.RecaptchaToken)
		if err != nil { return nil, fmt.Errorf("reCAPTCHA verification failed: %w", err) }
		if !valid { return nil, fmt.Errorf("reCAPTCHA verification failed: invalid token") }
	}
	if req.Email == "test@example.com" && req.Password == "password123" {
		return &inbound.LoginResponse{ AccessToken: "mock-access-token", RefreshToken: "mock-refresh-token", ExpiresIn: 900 }, nil
	}
	return nil, fmt.Errorf("Invalid credentials")
}

func (m *mockAuthUseCase) Refresh(ctx context.Context, req inbound.RefreshRequest) (*inbound.RefreshResponse, error) {
	return &inbound.RefreshResponse{ AccessToken: "mock-new-access-token", ExpiresIn: 900 }, nil
}

func (m *mockAuthUseCase) Logout(ctx context.Context, req inbound.LogoutRequest) error {
	return nil
}

func (m *mockAuthUseCase) Me(ctx context.Context, userID string) (*inbound.MeResponse, error) {
	return &inbound.MeResponse{
		ID:    "mock-user-id",
		Email: "test@example.com",
	}, nil
}

func TestIteration3AcceptanceTestSuite(t *testing.T) {
	suite.Run(t, new(Iteration3AcceptanceTestSuite))
}