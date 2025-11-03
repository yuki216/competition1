package jwt

import (
	"testing"
	"time"

	"github.com/vobe/auth-service/infrastructure/config"
	"github.com/vobe/auth-service/application/port/outbound"
)

func TestJWTService(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		JWTAlgorithm:    "HS256",
		AccessTokenTTL:  3600, // 1 hour
		RefreshTokenTTL: 2592000,
	}

	service, err := NewJWTService(cfg)
	if err != nil {
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	t.Run("GenerateAccessToken", func(t *testing.T) {
		token, err := service.GenerateAccessToken(outbound.TokenClaims{UserID: "user123"})
		if err != nil {
			t.Errorf("Failed to generate access token: %v", err)
		}
		if token == "" {
			t.Error("Access token should not be empty")
		}
	})

	t.Run("GenerateRefreshToken", func(t *testing.T) {
		token, err := service.GenerateRefreshToken()
		if err != nil {
			t.Errorf("Failed to generate refresh token: %v", err)
		}
		if token == "" {
			t.Error("Refresh token should not be empty")
		}
	})

	t.Run("ValidateAccessToken", func(t *testing.T) {
		tokenString, err := service.GenerateAccessToken(outbound.TokenClaims{UserID: "user123"})
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := service.ValidateAccessToken(tokenString)
		if err != nil {
			t.Errorf("Failed to validate token: %v", err)
		}
		if claims != nil && claims.UserID != "user123" {
			t.Errorf("Expected user ID 'user123', got '%s'", claims.UserID)
		}
	})

	t.Run("ValidateInvalidToken", func(t *testing.T) {
		_, err := service.ValidateAccessToken("invalid-token")
		if err == nil {
			t.Error("Should fail to validate invalid token")
		}
	})

	t.Run("ValidateExpiredToken", func(t *testing.T) {
		// Create service with very short TTL
		shortCfg := &config.Config{
			JWTSecret:       "test-secret",
			JWTAlgorithm:    "HS256",
			AccessTokenTTL:  1, // 1 second
			RefreshTokenTTL: 2592000,
		}
		
		shortService, err := NewJWTService(shortCfg)
		if err != nil {
			t.Fatalf("Failed to create JWT service: %v", err)
		}

		// Generate token
		token, err := shortService.GenerateAccessToken(outbound.TokenClaims{UserID: "user123"})
		if err != nil {
			t.Fatalf("Failed to generate access token: %v", err)
		}

		// Wait for token to expire
		time.Sleep(2 * time.Second)

		// Validate expired token
		_, err = shortService.ValidateAccessToken(token)
		if err == nil {
			t.Error("Should fail to validate expired token")
		}
	})
}