package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fixora/fixora/application/port/inbound"
	"github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain/entity"
	"github.com/fixora/fixora/infrastructure/service/logger"
	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepository         outbound.UserRepository
	refreshTokenRepository outbound.RefreshTokenRepository
	tokenService           outbound.TokenService
	passwordService        inbound.PasswordService
	recaptchaService       inbound.RecaptchaService
	rateLimitService       inbound.RateLimitService
	logger                 logger.Logger
	accessTokenTTL         time.Duration
	refreshTokenTTL        time.Duration
}

func NewAuthUseCase(
	userRepo outbound.UserRepository,
	refreshTokenRepo outbound.RefreshTokenRepository,
	tokenService outbound.TokenService,
	passwordService inbound.PasswordService,
	recaptchaService inbound.RecaptchaService,
	rateLimitService inbound.RateLimitService,
	logger logger.Logger,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) inbound.AuthUseCase {
	return &AuthUseCase{
		userRepository:         userRepo,
		refreshTokenRepository: refreshTokenRepo,
		tokenService:           tokenService,
		passwordService:        passwordService,
		recaptchaService:       recaptchaService,
		rateLimitService:       rateLimitService,
		logger:                 logger,
		accessTokenTTL:         accessTokenTTL,
		refreshTokenTTL:        refreshTokenTTL,
	}
}

func (uc *AuthUseCase) Login(ctx context.Context, req inbound.LoginRequest) (*inbound.LoginResponse, error) {
	// Log login attempt
	logger.LogAuthEvent(ctx, uc.logger, "login_attempt", "", "", true, map[string]interface{}{
		"email": req.Email,
	})

	// Validate request
	if err := uc.validateLoginRequest(req); err != nil {
		logger.LogAuthEvent(ctx, uc.logger, "login_validation_failed", "", "", false, map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
		})
		return nil, err
	}

	// Check rate limiting untuk IP
	ip := getClientIP(ctx)
	if uc.rateLimitService != nil {
		// Check if IP is blocked
		isBlocked, err := uc.rateLimitService.IsBlocked(ctx, fmt.Sprintf("ip:%s", ip))
		if err != nil {
			uc.logger.Error(ctx, "Failed to check IP block status", err, map[string]interface{}{
				"ip": ip,
			})
		}
		if isBlocked {
			logger.LogSecurityEvent(ctx, uc.logger, "blocked_ip_login_attempt", "MEDIUM", map[string]interface{}{
				"ip":    ip,
				"email": req.Email,
			})
			return nil, fmt.Errorf("IP address is blocked due to too many failed attempts")
		}

		// Check rate limit
		allowed, err := uc.rateLimitService.CheckLimit(ctx, fmt.Sprintf("ip:%s", ip), 5, 15*time.Minute)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check rate limit", err, map[string]interface{}{
				"ip": ip,
			})
		}
		if !allowed {
			// Block IP
			uc.rateLimitService.Block(ctx, fmt.Sprintf("ip:%s", ip), 30*time.Minute, "Rate limit exceeded")
			logger.LogSecurityEvent(ctx, uc.logger, "ip_rate_limit_exceeded", "HIGH", map[string]interface{}{
				"ip":    ip,
				"email": req.Email,
			})
			return nil, fmt.Errorf("Too many login attempts. Please try again later.")
		}
	}

	// Validate reCAPTCHA jika enabled
	if uc.recaptchaService != nil && uc.recaptchaService.IsEnabled() {
		start := time.Now()
		valid, err := uc.recaptchaService.VerifyToken(ctx, req.RecaptchaToken)
		duration := time.Since(start)

		logger.LogPerformance(ctx, uc.logger, "recaptcha_verification", duration, map[string]interface{}{
			"email": req.Email,
		})

		if err != nil {
			logger.LogSecurityEvent(ctx, uc.logger, "recaptcha_verification_failed", "MEDIUM", map[string]interface{}{
				"email": req.Email,
				"error": err.Error(),
			})
			return nil, fmt.Errorf("reCAPTCHA verification failed: %w", err)
		}
		if !valid {
			logger.LogSecurityEvent(ctx, uc.logger, "invalid_recaptcha_token", "MEDIUM", map[string]interface{}{
				"email": req.Email,
			})
			return nil, fmt.Errorf("Invalid reCAPTCHA token")
		}
	}

	// Find user by email
	user, err := uc.userRepository.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, outbound.ErrUserNotFound) {
			// Increment failed attempt counter
			if uc.rateLimitService != nil {
				uc.rateLimitService.Increment(ctx, fmt.Sprintf("ip:%s", ip), 15*time.Minute)
			}
			logger.LogAuthEvent(ctx, uc.logger, "login_failed_user_not_found", "", ip, false, map[string]interface{}{
				"email": req.Email,
			})
			return nil, fmt.Errorf("Invalid credentials")
		}
		uc.logger.Error(ctx, "Failed to find user", err, map[string]interface{}{
			"email": req.Email,
		})
		return nil, fmt.Errorf("Failed to find user: %w", err)
	}

	// Additional nil check for safety
	if user == nil {
		uc.logger.Error(ctx, "User returned as nil", nil, map[string]interface{}{
			"email": req.Email,
		})
		return nil, fmt.Errorf("Invalid credentials")
	}

	// Check user rate limiting
	if uc.rateLimitService != nil {
		isUserBlocked, err := uc.rateLimitService.IsBlocked(ctx, fmt.Sprintf("user:%s", user.ID))
		if err != nil {
			uc.logger.Error(ctx, "Failed to check user block status", err, map[string]interface{}{
				"user_id": user.ID,
			})
		}
		if isUserBlocked {
			logger.LogSecurityEvent(ctx, uc.logger, "blocked_user_login_attempt", "MEDIUM", map[string]interface{}{
				"user_id": user.ID,
				"email":   req.Email,
			})
			return nil, fmt.Errorf("User account is blocked due to too many failed attempts")
		}
	}

	// Verify password
	start := time.Now()
	isValid, err := uc.passwordService.VerifyPassword(req.Password, user.Password)
	duration := time.Since(start)

	logger.LogPerformance(ctx, uc.logger, "password_verification", duration, map[string]interface{}{
		"user_id": user.ID,
	})

	if err != nil {
		uc.logger.Error(ctx, "Password verification error", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("Password verification failed")
	}
	if !isValid {
		// Increment failed attempt counter
		if uc.rateLimitService != nil {
			uc.rateLimitService.Increment(ctx, fmt.Sprintf("ip:%s", ip), 15*time.Minute)
			uc.rateLimitService.Increment(ctx, fmt.Sprintf("user:%s", user.ID), 1*time.Hour)
		}
		logger.LogAuthEvent(ctx, uc.logger, "login_failed_invalid_password", user.ID, ip, false, map[string]interface{}{
			"email": req.Email,
		})
		return nil, fmt.Errorf("Invalid credentials")
	}

	// Generate tokens
	start = time.Now()
	accessToken, err := uc.tokenService.GenerateAccessToken(outbound.TokenClaims{UserID: user.ID, Email: user.Email, Role: user.Role})
	duration = time.Since(start)

	logger.LogPerformance(ctx, uc.logger, "access_token_generation", duration, map[string]interface{}{
		"user_id": user.ID,
	})

	if err != nil {
		uc.logger.Error(ctx, "Failed to generate access token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("Failed to generate access token: %w", err)
	}

	refreshToken, err := uc.tokenService.GenerateRefreshToken()
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate refresh token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("Failed to generate refresh token: %w", err)
	}

	// Determine refresh TTL dynamically based on RememberMe
	refreshTTL := uc.refreshTokenTTL
	// Default short TTL if RememberMe is false: use half of configured TTL (fallback 7 days if 30 days configured)
	if !req.RememberMe {
		// If configured TTL >= 14 days, make short TTL 7 days; else use half
		if refreshTTL >= (14 * 24 * time.Hour) {
			refreshTTL = 7 * 24 * time.Hour
		} else {
			refreshTTL = refreshTTL / 2
		}
	}

	expiresAt := time.Now().Add(refreshTTL)

	// Create refresh token entity
	refreshTokenEntity := entity.NewRefreshToken(
		generateID(),
		user.ID,
		refreshToken,
		expiresAt,
	)

	// Store refresh token
	if err := uc.refreshTokenRepository.Create(ctx, refreshTokenEntity); err != nil {
		uc.logger.Error(ctx, "Failed to create refresh token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("Failed to create refresh token: %w", err)
	}

	// Reset failed attempts on successful login
	if uc.rateLimitService != nil {
		// Reset IP attempts (implement reset logic if needed)
		uc.logger.Info(ctx, "Login successful, resetting rate limit counters", map[string]interface{}{
			"user_id": user.ID,
			"ip":      ip,
		})
	}

	logger.LogAuthEvent(ctx, uc.logger, "login_successful", user.ID, ip, true, map[string]interface{}{
		"email":               req.Email,
		"remember_me":         req.RememberMe,
		"refresh_ttl_seconds": int(refreshTTL.Seconds()),
	})

	return &inbound.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int(uc.accessTokenTTL.Seconds()),
		RefreshExpiresIn: int(refreshTTL.Seconds()),
		User: inbound.MeResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (uc *AuthUseCase) Refresh(ctx context.Context, req inbound.RefreshRequest) (*inbound.RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	// Find refresh token
	refreshTokenEntity, err := uc.refreshTokenRepository.FindByToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, outbound.ErrRefreshTokenNotFound) {
			logger.LogSecurityEvent(ctx, uc.logger, "refresh_token_not_found", "MEDIUM", map[string]interface{}{
				"token": "[REDACTED]",
			})
			return nil, fmt.Errorf("invalid refresh token")
		}
		uc.logger.Error(ctx, "Failed to find refresh token", err, map[string]interface{}{
			"token": "[REDACTED]",
		})
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Validate refresh token
	if refreshTokenEntity.IsExpired() {
		logger.LogSecurityEvent(ctx, uc.logger, "refresh_token_expired", "MEDIUM", map[string]interface{}{
			"user_id": refreshTokenEntity.UserID,
		})
		return nil, fmt.Errorf("refresh token expired")
	}

	if refreshTokenEntity.IsRevoked() {
		logger.LogSecurityEvent(ctx, uc.logger, "refresh_token_revoked", "HIGH", map[string]interface{}{
			"user_id": refreshTokenEntity.UserID,
		})
		return nil, fmt.Errorf("refresh token revoked")
	}

	// Revoke current refresh token
	if err := uc.refreshTokenRepository.Revoke(ctx, req.RefreshToken); err != nil {
		uc.logger.Error(ctx, "Failed to revoke refresh token", err, map[string]interface{}{
			"user_id": refreshTokenEntity.UserID,
		})
		return nil, fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// Find user
	user, err := uc.userRepository.FindByID(ctx, refreshTokenEntity.UserID)
	if err != nil {
		if errors.Is(err, outbound.ErrUserNotFound) {
			logger.LogSecurityEvent(ctx, uc.logger, "refresh_user_not_found", "HIGH", map[string]interface{}{
				"user_id": refreshTokenEntity.UserID,
			})
			return nil, fmt.Errorf("user not found")
		}
		uc.logger.Error(ctx, "Failed to find user", err, map[string]interface{}{
			"user_id": refreshTokenEntity.UserID,
		})
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new tokens
	accessToken, err := uc.tokenService.GenerateAccessToken(outbound.TokenClaims{UserID: user.ID, Email: user.Email, Role: user.Role})
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate access token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := uc.tokenService.GenerateRefreshToken()
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate refresh token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create new refresh token entity
	newRefreshTokenEntity := entity.NewRefreshToken(
		generateID(),
		user.ID,
		newRefreshToken,
		time.Now().Add(uc.refreshTokenTTL),
	)

	// Store new refresh token
	if err := uc.refreshTokenRepository.Create(ctx, newRefreshTokenEntity); err != nil {
		uc.logger.Error(ctx, "Failed to create refresh token", err, map[string]interface{}{
			"user_id": user.ID,
		})
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	logger.LogAuthEvent(ctx, uc.logger, "token_refresh_successful", user.ID, "", true, map[string]interface{}{
		"user_id": user.ID,
	})

	return &inbound.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(uc.accessTokenTTL.Seconds()),
	}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, req inbound.LogoutRequest) error {
	if req.RefreshToken != "" {
		// Revoke specific refresh token
		if err := uc.refreshTokenRepository.Revoke(ctx, req.RefreshToken); err != nil {
			if errors.Is(err, outbound.ErrRefreshTokenNotFound) {
				logger.LogAuthEvent(ctx, uc.logger, "logout_token_not_found", "", "", false, map[string]interface{}{
					"token": "[REDACTED]",
				})
				return fmt.Errorf("token not found")
			}
			uc.logger.Error(ctx, "Failed to revoke refresh token", err, map[string]interface{}{
				"token": "[REDACTED]",
			})
			return fmt.Errorf("failed to revoke refresh token: %w", err)
		}

		logger.LogAuthEvent(ctx, uc.logger, "logout_successful", "", "", true, map[string]interface{}{
			"token": "[REDACTED]",
		})
		return nil
	}

	// If no specific refresh token provided, revoke all refresh tokens for user via access token claims
	if req.UserID != "" {
		if err := uc.refreshTokenRepository.RevokeByUserID(ctx, req.UserID); err != nil {
			uc.logger.Error(ctx, "Failed to revoke refresh tokens by user", err, map[string]interface{}{
				"user_id": req.UserID,
			})
			return fmt.Errorf("failed to revoke refresh tokens by user: %w", err)
		}

		logger.LogAuthEvent(ctx, uc.logger, "logout_successful", req.UserID, "", true, map[string]interface{}{})
		return nil
	}

	return fmt.Errorf("access token required")
}

func (uc *AuthUseCase) Me(ctx context.Context, userID string) (*inbound.MeResponse, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Find user
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, outbound.ErrUserNotFound) {
			logger.LogSecurityEvent(ctx, uc.logger, "me_user_not_found", "MEDIUM", map[string]interface{}{
				"user_id": userID,
			})
			return nil, fmt.Errorf("user not found")
		}
		uc.logger.Error(ctx, "Failed to find user", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	logger.LogAuthEvent(ctx, uc.logger, "me_request_successful", user.ID, "", true, map[string]interface{}{
		"user_id": userID,
	})

	return &inbound.MeResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

func (uc *AuthUseCase) validateLoginRequest(req inbound.LoginRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// Helper functions

func generateID() string {
	return uuid.New().String()
}

func getClientIP(ctx context.Context) string {
	// This is a simplified implementation
	// In real implementation, get IP from request context
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return "unknown"
}
