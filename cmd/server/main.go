package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/sirupsen/logrus"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/usecase/user_management"
	"github.com/fixora/fixora/infrastructure/adapter/postgres"
	"github.com/fixora/fixora/infrastructure/config"
	"github.com/fixora/fixora/infrastructure/http/handler"
	"github.com/fixora/fixora/infrastructure/http/middleware"
	"github.com/fixora/fixora/infrastructure/persistence/usecase"
	"github.com/fixora/fixora/infrastructure/service/jwt"
	"github.com/fixora/fixora/infrastructure/service/logger"
	"github.com/fixora/fixora/infrastructure/service/password"
	"github.com/fixora/fixora/infrastructure/service/ratelimit"
	"github.com/fixora/fixora/infrastructure/service/recaptcha"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logger
	structuredLogger := logger.NewStructuredLogger(logger.LoggerConfig{
		Level:               cfg.LogLevel,
		Format:              cfg.LogFormat,
		CorrelationIDHeader: middleware.CorrelationIDHeader,
		EnableRequestLog:    cfg.LogEnableRequestLog,
		EnableResponseLog:   cfg.LogEnableResponseLog,
		ServiceName:         "auth-service",
	})
	structuredLogger.Info(ctx, "Application starting", map[string]interface{}{
		"version": "1.0.0",
		"env":     cfg.Environment,
	})

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		structuredLogger.Error(ctx, "Failed to connect to database", err, map[string]interface{}{
			"database_url": cfg.DatabaseURL,
		})
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		structuredLogger.Error(ctx, "Failed to ping database", err, map[string]interface{}{
			"database_url": cfg.DatabaseURL,
		})
		log.Fatalf("Failed to ping database: %v", err)
	}
	structuredLogger.Info(ctx, "Database connection established", map[string]interface{}{
		"database_url": cfg.DatabaseURL,
	})

	// Initialize rate limiting service (Redis-backed or noop based on config)
	var rateLimitService inbound.RateLimitService
	{
		rlConfig := ratelimit.RateLimitConfig{
			Enabled:       cfg.RateLimitEnabled,
			RedisURL:      cfg.RedisURL,
			IPAttempts:    cfg.RateLimitIPAttempts,
			IPWindow:      cfg.RateLimitIPWindow,
			UserAttempts:  cfg.RateLimitUserAttempts,
			UserWindow:    cfg.RateLimitUserWindow,
			BlockDuration: cfg.RateLimitBlockDuration,
		}
		rlLogger := logrus.New()
		rs, err := ratelimit.NewRateLimitService(rlConfig, rlLogger)
		if err != nil {
			structuredLogger.Error(ctx, "Failed to initialize rate limit service", err, map[string]interface{}{
				"redis_url": cfg.RedisURL,
			})
		} else {
			// Bridge ratelimit service into inbound interface
			if s, ok := rs.(inbound.RateLimitService); ok {
				rateLimitService = s
				structuredLogger.Info(ctx, "Rate limiting service initialized", map[string]interface{}{
					"redis_url": cfg.RedisURL,
					"enabled":   cfg.RateLimitEnabled,
				})
			} else {
				structuredLogger.Warn(ctx, "Rate limit service does not satisfy inbound interface", map[string]interface{}{})
			}
		}
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepositoryAdapter(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepositoryAdapter(db, cfg.RefreshTokenSalt)

	// Initialize services
	tokenService, err := jwt.NewJWTService(cfg)
	if err != nil {
		structuredLogger.Error(ctx, "Failed to initialize JWT service", err, map[string]interface{}{
			"config": cfg,
		})
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}
	passwordService := password.NewBcryptPasswordService(10)

	// Initialize reCAPTCHA service
	var recaptchaService inbound.RecaptchaService
	if cfg.RecaptchaEnabled {
		recaptchaService = recaptcha.NewRecaptchaService(
			cfg.RecaptchaSecret,
			cfg.RecaptchaEnabled,
			cfg.RecaptchaSkip,
			cfg.RecaptchaTimeout,
			structuredLogger,
		)
		structuredLogger.Info(ctx, "reCAPTCHA service initialized", map[string]interface{}{
			"enabled":  cfg.RecaptchaEnabled,
			"site_key": cfg.RecaptchaSiteKey,
		})
	} else {
		recaptchaService = recaptcha.NewNoopRecaptchaService(structuredLogger)
		structuredLogger.Info(ctx, "reCAPTCHA disabled", map[string]interface{}{})
	}

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		refreshTokenRepo,
		tokenService,
		passwordService,
		recaptchaService,
		rateLimitService,
		structuredLogger,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	userManagementUseCase := user_management.NewUserManagementUseCase(userRepo, passwordService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, structuredLogger)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	userManagementHandler := handler.NewUserManagementHandler(userManagementUseCase, authMiddleware)

	// Setup routes
	mux := http.NewServeMux()
	mux.Handle("/v1/auth/login", rateLimitMiddleware.RateLimit(http.HandlerFunc(authHandler.Login)))
	mux.Handle("/v1/auth/refresh", rateLimitMiddleware.RateLimit(http.HandlerFunc(authHandler.Refresh)))
	mux.HandleFunc("/v1/auth/logout", authMiddleware.RequireAuth(authHandler.Logout))
	mux.HandleFunc("/v1/auth/me", authMiddleware.RequireAuth(authHandler.Me))

	// Register user management routes
	userManagementHandler.RegisterRoutes(mux)

	// Swagger UI & OpenAPI docs under /docs
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/swagger-ui.html")
	})
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("api"))))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	// Create server
	// Compose middleware: CorrelationID then CORS (if enabled)
	handler := middleware.CorrelationIDMiddleware(mux)
	if cfg.CORSEnabled && len(cfg.CORSAllowedOrigins) > 0 {
		handler = middleware.CORSMiddleware(handler, cfg.CORSAllowedOrigins, cfg.CORSAllowCredentials)
	}
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		structuredLogger.Info(ctx, "Starting server", map[string]interface{}{
			"host": cfg.ServerHost,
			"port": cfg.ServerPort,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			structuredLogger.Error(ctx, "Server failed to start", err, map[string]interface{}{
				"host": cfg.ServerHost,
				"port": cfg.ServerPort,
			})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	structuredLogger.Info(ctx, "Shutting down server...", map[string]interface{}{})

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		structuredLogger.Error(ctx, "Server forced to shutdown", err, map[string]interface{}{})
	}
	structuredLogger.Info(ctx, "Server exited", map[string]interface{}{})
}
