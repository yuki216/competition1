package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/vobe/auth-service/application/port/outbound"
	"github.com/vobe/auth-service/infrastructure/http/response"
)

const (
	AuthUserKey = "auth_user"
)

type AuthMiddleware struct {
	tokenService outbound.TokenService
}

func NewAuthMiddleware(tokenService outbound.TokenService) *AuthMiddleware {
	return &AuthMiddleware{
		tokenService: tokenService,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Authorization header required")
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(w, "Invalid authorization header format")
			return
		}

		token := parts[1]
		if token == "" {
			response.Unauthorized(w, "Token cannot be empty")
			return
		}

		// Validate token
		claims, err := m.tokenService.ValidateAccessToken(token)
		if err != nil {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), AuthUserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header, proceed without user context
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, proceed without user context
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]
		if token == "" {
			// Empty token, proceed without user context
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate token
		claims, err := m.tokenService.ValidateAccessToken(token)
		if err != nil {
			// Invalid token, proceed without user context
			next.ServeHTTP(w, r)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), AuthUserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdmin ensures that the user has admin role
func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First require authentication
		authHandler := m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			// Get user claims from context
			claims := GetUserClaims(r.Context())
			if claims == nil {
				response.Unauthorized(w, "User not authenticated")
				return
			}

			// Check if user has admin role
			if !isAdminRole(claims.Role) {
				response.Forbidden(w, "Admin access required")
				return
			}

			// User is admin, proceed
			next.ServeHTTP(w, r)
		})

		authHandler.ServeHTTP(w, r)
	}
}

// isAdminRole checks if the role is considered admin
func isAdminRole(role string) bool {
	adminRoles := []string{"admin", "superadmin"}
	for _, adminRole := range adminRoles {
		if role == adminRole {
			return true
		}
	}
	return false
}

// GetUserClaims retrieves user claims from context
func GetUserClaims(ctx context.Context) *outbound.TokenClaims {
	if claims, ok := ctx.Value(AuthUserKey).(*outbound.TokenClaims); ok {
		return claims
	}
	return nil
}