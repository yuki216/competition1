package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/infrastructure/http/response"
	"github.com/fixora/fixora/infrastructure/service/logger"
)

type RateLimitMiddleware struct {
	rateLimitService inbound.RateLimitService
	logger           logger.Logger
}

func NewRateLimitMiddleware(rateLimitService inbound.RateLimitService, logger logger.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimitService: rateLimitService,
		logger:           logger,
	}
}

func (m *RateLimitMiddleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		clientIP := getClientIP(r)

		// Skip rate limiting if service is not available
		if m.rateLimitService == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Create rate limit key based on endpoint and IP
		var key string
		var limit int
		var window time.Duration

		switch {
		case strings.Contains(r.URL.Path, "/login"):
			key = fmt.Sprintf("login:ip:%s", clientIP)
			limit = 10 // 10 login attempts per 15 minutes
			window = 15 * time.Minute
		case strings.Contains(r.URL.Path, "/refresh"):
			key = fmt.Sprintf("refresh:ip:%s", clientIP)
			limit = 30 // 30 refresh attempts per hour
			window = 1 * time.Hour
		default:
			key = fmt.Sprintf("general:ip:%s", clientIP)
			limit = 100 // 100 requests per minute for general endpoints
			window = 1 * time.Minute
		}

		// Check if IP is blocked
		isBlocked, err := m.rateLimitService.IsBlocked(ctx, key)
		if err != nil {
			m.logger.Error(ctx, "Failed to check block status", err, map[string]interface{}{
				"ip":  clientIP,
				"key": key,
			})
			// Continue with request on error
		}

		if isBlocked {
			logger.LogSecurityEvent(ctx, m.logger, "rate_limit_blocked", "MEDIUM", map[string]interface{}{
				"ip":        clientIP,
				"path":      r.URL.Path,
				"key":       key,
				"userAgent": r.UserAgent(),
			})

			w.Header().Set("Retry-After", "900") // 15 minutes
			response.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
			return
		}

		// Check rate limit
		allowed, err := m.rateLimitService.CheckLimit(ctx, key, limit, window)
		if err != nil {
			m.logger.Error(ctx, "Failed to check rate limit", err, map[string]interface{}{
				"ip":  clientIP,
				"key": key,
			})
			// Continue with request on error
		}

		if !allowed {
			// Block the IP
			blockDuration := 15 * time.Minute
			if strings.Contains(r.URL.Path, "/login") {
				blockDuration = 30 * time.Minute // Longer block for login attempts
			}

			err := m.rateLimitService.Block(ctx, key, blockDuration, "Rate limit exceeded")
			if err != nil {
				m.logger.Error(ctx, "Failed to block IP", err, map[string]interface{}{
					"ip":  clientIP,
					"key": key,
				})
			}

			logger.LogSecurityEvent(ctx, m.logger, "rate_limit_exceeded", "HIGH", map[string]interface{}{
				"ip":        clientIP,
				"path":      r.URL.Path,
				"key":       key,
				"userAgent": r.UserAgent(),
			})

			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(blockDuration.Seconds())))
			response.Error(w, http.StatusTooManyRequests, "Too many requests. Please try again later.")
			return
		}

		// Log rate limit status
		attempts, err := m.rateLimitService.GetAttempts(ctx, key)
		if err != nil {
			m.logger.Error(ctx, "Failed to get attempts", err, map[string]interface{}{
				"ip":  clientIP,
				"key": key,
			})
		} else {
			logger.LogPerformance(ctx, m.logger, "rate_limit_check", time.Duration(0), map[string]interface{}{
				"ip":       clientIP,
				"path":     r.URL.Path,
				"key":      key,
				"attempts": attempts,
				"limit":    limit,
			})
		}

		// Continue with request
		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts client IP from request
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
