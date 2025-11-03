package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/infrastructure/http/middleware"
	"github.com/vobe/auth-service/infrastructure/http/response"
	"github.com/vobe/auth-service/infrastructure/http/validator"
)

type AuthHandler struct {
	authUseCase inbound.AuthUseCase
}

func NewAuthHandler(authUseCase inbound.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

type LoginRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	RememberMe   bool   `json:"remember_me"`
	RecaptchaToken string `json:"recaptcha_token"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Validate email
	if !validator.ValidateEmail(req.Email) {
		response.UnprocessableEntity(w, "Invalid email format")
		return
	}

	// Validate password
	if !validator.ValidateRequired(req.Password) {
		response.UnprocessableEntity(w, "Password is required")
		return
	}

	// Validate reCAPTCHA token if provided (optional for now)
	if req.RecaptchaToken != "" {
		if !validator.ValidateRequired(req.RecaptchaToken) {
			response.UnprocessableEntity(w, "reCAPTCHA token is required")
			return
		}
	}

	// Get client IP
	clientIP := getClientIP(r)
	
	// Create context with client IP
	ctx := context.WithValue(r.Context(), "client_ip", clientIP)

	// Call use case
	loginReq := inbound.LoginRequest{
		Email:          req.Email,
		Password:       req.Password,
		RememberMe:     req.RememberMe,
		RecaptchaToken: req.RecaptchaToken,
	}

	loginRes, err := h.authUseCase.Login(ctx, loginReq)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "blocked"):
			response.Error(w, http.StatusTooManyRequests, err.Error())
		case strings.Contains(err.Error(), "reCAPTCHA"):
			response.Error(w, http.StatusBadRequest, err.Error())
		case strings.Contains(err.Error(), "Invalid credentials"):
			response.Unauthorized(w, "Email atau password salah")
		case strings.Contains(err.Error(), "user not found"):
			response.Unauthorized(w, "Email atau password salah")
		case strings.Contains(err.Error(), "Too many"):
			response.Error(w, http.StatusTooManyRequests, err.Error())
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	// Set refresh token cookie with dynamic TTL
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    loginRes.RefreshToken,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   loginRes.RefreshExpiresIn,
	})

	// Return success response
	response.Success(w, http.StatusOK, "success", LoginResponse{
		AccessToken: loginRes.AccessToken,
		ExpiresIn:   loginRes.ExpiresIn,
	})
}

// Helper function to get client IP
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if ip != "" {
		// Remove port if present
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}

	return ip
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get refresh token from cookie or header
	var refreshToken string
	
	// Try cookie first
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	} else {
		// Try header
		refreshToken = r.Header.Get("Refresh-Token")
	}

	if refreshToken == "" {
		response.Unauthorized(w, "Refresh token required")
		return
	}

	// Call use case
	refreshReq := inbound.RefreshRequest{
		RefreshToken: refreshToken,
	}

	refreshRes, err := h.authUseCase.Refresh(r.Context(), refreshReq)
	if err != nil {
		switch err.Error() {
		case "token invalid", "token expired", "token revoked", "invalid refresh token", "refresh token expired", "refresh token revoked":
			response.Unauthorized(w, "Invalid or expired refresh token")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	// Return success response
	response.Success(w, http.StatusOK, "success", LoginResponse{
		AccessToken: refreshRes.AccessToken,
		ExpiresIn:   refreshRes.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    // Get user claims from access token via middleware
    claims := middleware.GetUserClaims(r.Context())
    if claims == nil || claims.UserID == "" {
        response.Unauthorized(w, "Authorization header required")
        return
    }

    // Call use case with userID to revoke all refresh tokens for this user
    logoutReq := inbound.LogoutRequest{ UserID: claims.UserID }
    if err := h.authUseCase.Logout(r.Context(), logoutReq); err != nil {
        // Map common errors
        switch err.Error() {
        case "token not found":
            // For user-level revoke, this case is unlikely; still handle
            response.Unauthorized(w, "Invalid refresh token")
        case "access token required":
            response.Unauthorized(w, "Authorization header required")
        default:
            response.InternalServerError(w, "Internal server error")
        }
        return
    }

    // Clear refresh token cookie if present
    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    "",
        Path:     "/v1/auth/refresh",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   -1,
    })

    // Return 204 No Content
    w.WriteHeader(http.StatusNoContent)
    return
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get user claims from context
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		response.Unauthorized(w, "User not authenticated")
		return
	}

	// Call use case
	meRes, err := h.authUseCase.Me(r.Context(), claims.UserID)
	if err != nil {
		switch err.Error() {
		case "user not found":
			response.NotFound(w, "User not found")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	// Return success response
	response.Success(w, http.StatusOK, "success", meRes)
}