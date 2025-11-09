package error

import (
	"errors"
	"fmt"
)

// ErrorCode represents a unique error code
type ErrorCode string

// Error codes for different categories
const (
	// Authentication Errors (1xxx)
	ErrCodeInvalidCredentials     ErrorCode = "AUTH_1001"
	ErrCodeUserNotFound          ErrorCode = "AUTH_1002"
	ErrCodeInvalidToken          ErrorCode = "AUTH_1003"
	ErrCodeTokenExpired          ErrorCode = "AUTH_1004"
	ErrCodeTokenRevoked          ErrorCode = "AUTH_1005"
	ErrCodeInvalidRefreshToken   ErrorCode = "AUTH_1006"
	ErrCodeRefreshTokenExpired   ErrorCode = "AUTH_1007"
	ErrCodeRefreshTokenRevoked   ErrorCode = "AUTH_1008"

	// Validation Errors (2xxx)
	ErrCodeInvalidEmail          ErrorCode = "VALID_2001"
	ErrCodeInvalidPassword       ErrorCode = "VALID_2002"
	ErrCodeMissingEmail          ErrorCode = "VALID_2003"
	ErrCodeMissingPassword       ErrorCode = "VALID_2004"
	ErrCodeInvalidRequest        ErrorCode = "VALID_2005"

	// Rate Limiting Errors (3xxx)
	ErrCodeRateLimitExceeded     ErrorCode = "RATE_3001"
	ErrCodeIPBlocked             ErrorCode = "RATE_3002"
	ErrCodeUserBlocked           ErrorCode = "RATE_3003"
	ErrCodeTooManyAttempts       ErrorCode = "RATE_3004"

	// reCAPTCHA Errors (4xxx)
	ErrCodeRecaptchaInvalid      ErrorCode = "RECAPTCHA_4001"
	ErrCodeRecaptchaFailed       ErrorCode = "RECAPTCHA_4002"
	ErrCodeRecaptchaTimeout      ErrorCode = "RECAPTCHA_4003"
	ErrCodeRecaptchaMissing      ErrorCode = "RECAPTCHA_4004"

	// Database Errors (5xxx)
	ErrCodeDatabaseError         ErrorCode = "DB_5001"
	ErrCodeUserCreationFailed    ErrorCode = "DB_5002"
	ErrCodeTokenCreationFailed     ErrorCode = "DB_5003"
	ErrCodeTokenUpdateFailed     ErrorCode = "DB_5004"

	// Server Errors (6xxx)
	ErrCodeInternalServerError   ErrorCode = "SERVER_6001"
	ErrCodeServiceUnavailable    ErrorCode = "SERVER_6002"
	ErrCodeConfigurationError    ErrorCode = "SERVER_6003"
	ErrCodeExternalServiceError  ErrorCode = "SERVER_6004"

	// Security Errors (7xxx)
	ErrCodeSecurityViolation     ErrorCode = "SEC_7001"
	ErrCodeSuspiciousActivity    ErrorCode = "SEC_7002"
	ErrCodeUnauthorizedAccess    ErrorCode = "SEC_7003"
)

// AppError represents a structured application error
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the cause error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, details string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// Common error constructors

// Authentication errors
func ErrInvalidCredentials(details string) *AppError {
	return NewAppError(ErrCodeInvalidCredentials, "Invalid email or password", details, nil)
}

func ErrUserNotFound(userID string) *AppError {
	return NewAppError(ErrCodeUserNotFound, "User not found", fmt.Sprintf("User ID: %s", userID), nil)
}

func ErrInvalidToken(details string) *AppError {
	return NewAppError(ErrCodeInvalidToken, "Invalid token", details, nil)
}

func ErrTokenExpired(details string) *AppError {
	return NewAppError(ErrCodeTokenExpired, "Token has expired", details, nil)
}

func ErrTokenRevoked(details string) *AppError {
	return NewAppError(ErrCodeTokenRevoked, "Token has been revoked", details, nil)
}

// Validation errors
func ErrInvalidEmail(email string) *AppError {
	return NewAppError(ErrCodeInvalidEmail, "Invalid email format", fmt.Sprintf("Email: %s", email), nil)
}

func ErrInvalidPassword(details string) *AppError {
	return NewAppError(ErrCodeInvalidPassword, "Invalid password", details, nil)
}

func ErrMissingField(field string) *AppError {
	return NewAppError(ErrCodeInvalidRequest, "Missing required field", fmt.Sprintf("Field: %s", field), nil)
}

// Rate limiting errors
func ErrRateLimitExceeded(attempts int, window string) *AppError {
	return NewAppError(ErrCodeRateLimitExceeded, "Too many requests", fmt.Sprintf("Attempts: %d, Window: %s", attempts, window), nil)
}

func ErrIPBlocked(ip string, duration string) *AppError {
	return NewAppError(ErrCodeIPBlocked, "IP address is blocked", fmt.Sprintf("IP: %s, Duration: %s", ip, duration), nil)
}

func ErrUserBlocked(userID string, duration string) *AppError {
	return NewAppError(ErrCodeUserBlocked, "User account is blocked", fmt.Sprintf("User ID: %s, Duration: %s", userID, duration), nil)
}

// reCAPTCHA errors
func ErrRecaptchaInvalid(details string) *AppError {
	return NewAppError(ErrCodeRecaptchaInvalid, "Invalid reCAPTCHA token", details, nil)
}

func ErrRecaptchaFailed(details string) *AppError {
	return NewAppError(ErrCodeRecaptchaFailed, "reCAPTCHA verification failed", details, nil)
}

func ErrRecaptchaTimeout(timeout string) *AppError {
	return NewAppError(ErrCodeRecaptchaTimeout, "reCAPTCHA verification timeout", fmt.Sprintf("Timeout: %s", timeout), nil)
}

// Database errors
func ErrDatabaseError(operation string, cause error) *AppError {
	return NewAppError(ErrCodeDatabaseError, "Database operation failed", fmt.Sprintf("Operation: %s", operation), cause)
}

// Server errors
func ErrInternalServerError(details string, cause error) *AppError {
	return NewAppError(ErrCodeInternalServerError, "Internal server error", details, cause)
}

func ErrServiceUnavailable(service string) *AppError {
	return NewAppError(ErrCodeServiceUnavailable, "Service temporarily unavailable", fmt.Sprintf("Service: %s", service), nil)
}

func ErrConfigurationError(config string) *AppError {
	return NewAppError(ErrCodeConfigurationError, "Configuration error", fmt.Sprintf("Config: %s", config), nil)
}

// Security errors
func ErrSecurityViolation(details string) *AppError {
	return NewAppError(ErrCodeSecurityViolation, "Security violation detected", details, nil)
}

func ErrSuspiciousActivity(activity string) *AppError {
	return NewAppError(ErrCodeSuspiciousActivity, "Suspicious activity detected", fmt.Sprintf("Activity: %s", activity), nil)
}

// Error mapping for HTTP status codes
func GetHTTPStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		switch {
		case appErr.Code >= ErrCodeInvalidCredentials && appErr.Code < ErrCodeInvalidEmail:
			return 401 // Unauthorized
		case appErr.Code >= ErrCodeInvalidEmail && appErr.Code < ErrCodeRateLimitExceeded:
			return 400 // Bad Request
		case appErr.Code >= ErrCodeRateLimitExceeded && appErr.Code < ErrCodeRecaptchaInvalid:
			return 429 // Too Many Requests
		case appErr.Code >= ErrCodeRecaptchaInvalid && appErr.Code < ErrCodeDatabaseError:
			return 400 // Bad Request
		case appErr.Code >= ErrCodeDatabaseError && appErr.Code < ErrCodeInternalServerError:
			return 503 // Service Unavailable
		case appErr.Code >= ErrCodeInternalServerError && appErr.Code < ErrCodeSecurityViolation:
			return 500 // Internal Server Error
		case appErr.Code >= ErrCodeSecurityViolation:
			return 403 // Forbidden
		}
	}
	return 500 // Default to Internal Server Error
}

// Error response structure for API responses
type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   *AppError `json:"error"`
	TraceID string    `json:"trace_id,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err *AppError, traceID string) *ErrorResponse {
	return &ErrorResponse{
		Success: false,
		Error:   err,
		TraceID: traceID,
	}
}