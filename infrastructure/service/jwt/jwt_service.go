package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/infrastructure/config"
)

type JWTService struct {
	config        *config.Config
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	hmacSecret    []byte
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

func NewJWTService(cfg *config.Config) (*JWTService, error) {
	service := &JWTService{
		config: cfg,
	}

	switch cfg.JWTAlgorithm {
	case "HS256":
		service.hmacSecret = []byte(cfg.JWTSecret)
	case "RS256":
		// For now, we'll use HS256 fallback. In production, load RSA keys from files
		service.hmacSecret = []byte(cfg.JWTSecret)
	default:
		return nil, fmt.Errorf("unsupported JWT algorithm: %s", cfg.JWTAlgorithm)
	}

	return service, nil
}

func (s *JWTService) GenerateAccessToken(claims outbound.TokenClaims) (string, error) {
	tokenClaims := jwt.MapClaims{
		"user_id": claims.UserID,
		"exp":     time.Now().Add(s.config.AccessTokenTTL).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "access",
		}

	var token *jwt.Token
	if s.config.JWTAlgorithm == "HS256" {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	} else {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	}

	tokenString, err := token.SignedString(s.hmacSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

func (s *JWTService) GenerateRefreshToken() (string, error) {
	// Generate cryptographically secure random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base64 URL-safe string
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *JWTService) ValidateAccessToken(tokenString string) (*outbound.TokenClaims, error) {
	var claims jwt.MapClaims
	
	if s.config.JWTAlgorithm == "HS256" {
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.hmacSecret, nil
		})

		if err != nil {
			return nil, s.handleValidationError(err)
		}

		if !token.Valid {
			return nil, ErrInvalidToken
		}

		var ok bool
		claims, ok = token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, ErrInvalidToken
		}
	} else {
		return nil, fmt.Errorf("unsupported JWT algorithm for validation: %s", s.config.JWTAlgorithm)
	}

	// Extract claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Verify token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return nil, ErrInvalidToken
	}

	return &outbound.TokenClaims{
		UserID: userID,
	}, nil
}

func (s *JWTService) handleValidationError(err error) error {
	if errors.Is(err, jwt.ErrTokenExpired) {
		return ErrTokenExpired
	}
	if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		return ErrInvalidToken
	}
	return ErrInvalidToken
}