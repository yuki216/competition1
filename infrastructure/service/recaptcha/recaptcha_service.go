package recaptcha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/fixora/fixora/infrastructure/service/logger"
)

// RecaptchaService mendefinisikan interface untuk reCAPTCHA validation
type RecaptchaService interface {
	VerifyToken(ctx context.Context, token string) (bool, error)
	IsEnabled() bool
}

// recaptchaService implementasi RecaptchaService
type recaptchaService struct {
	secretKey     string
	siteVerifyURL string
	timeout       time.Duration
	enabled       bool
	skip          bool
	logger        logger.Logger
	httpClient    *http.Client
}

// RecaptchaResponse response dari Google reCAPTCHA API
type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score,omitempty"`
	Action      string    `json:"action,omitempty"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes,omitempty"`
}

// NewRecaptchaService membuat instance baru dari RecaptchaService
func NewRecaptchaService(secretKey string, enabled bool, skip bool, timeout time.Duration, log logger.Logger) RecaptchaService {
	return &recaptchaService{
		secretKey:     secretKey,
		siteVerifyURL: "https://www.google.com/recaptcha/api/siteverify",
		timeout:       timeout,
		enabled:       enabled,
		skip:          skip,
		logger:        log,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// VerifyToken memvalidasi reCAPTCHA token dengan Google API
func (s *recaptchaService) VerifyToken(ctx context.Context, token string) (bool, error) {
	// Jika reCAPTCHA di-skip atau disabled, return true
	if s.skip || !s.enabled {
		s.logger.Debug(ctx, "reCAPTCHA verification skipped", map[string]interface{}{})
		return true, nil
	}

	// Validasi input
	if token == "" {
		s.logger.Warn(ctx, "reCAPTCHA token is empty", map[string]interface{}{})
		return false, fmt.Errorf("reCAPTCHA token is required")
	}

	// Prepare request
	data := url.Values{}
	data.Set("secret", s.secretKey)
	data.Set("response", token)

	// Create request dengan timeout
	req, err := http.NewRequestWithContext(ctx, "POST", s.siteVerifyURL, nil)
	if err != nil {
		s.logger.Error(ctx, "failed to create reCAPTCHA request", err, map[string]interface{}{})
		return false, fmt.Errorf("failed to create reCAPTCHA request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(ctx, "reCAPTCHA request failed", err, map[string]interface{}{})
		return false, fmt.Errorf("reCAPTCHA service unavailable: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.logger.Error(ctx, "failed to decode reCAPTCHA response", err, map[string]interface{}{})
		return false, fmt.Errorf("failed to decode reCAPTCHA response: %w", err)
	}

	fields := map[string]interface{}{
		"success":     result.Success,
		"score":       result.Score,
		"action":      result.Action,
		"hostname":    result.Hostname,
		"error_codes": result.ErrorCodes,
	}

	if result.Success {
		s.logger.Info(ctx, "reCAPTCHA verification successful", fields)
		return true, nil
	}

	s.logger.Warn(ctx, "reCAPTCHA verification failed", fields)
	return false, fmt.Errorf("reCAPTCHA verification failed")
}

// IsEnabled mengembalikan status enabled dari service
func (s *recaptchaService) IsEnabled() bool {
	return s.enabled && !s.skip
}
