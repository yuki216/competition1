package recaptcha

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/fixora/fixora/infrastructure/service/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient untuk mocking HTTP calls
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestRecaptchaService_VerifyToken_Success(t *testing.T) {
	log := logger.NewStructuredLogger(logger.LoggerConfig{Level: "debug", Format: "text", ServiceName: "recaptcha-test"})

	service := &recaptchaService{
		secretKey:     "test-secret-key",
		siteVerifyURL: "https://www.google.com/recaptcha/api/siteverify",
		timeout:       5 * time.Second,
		enabled:       true,
		skip:          false,
		logger:        log,
		httpClient:    &http.Client{},
	}

	// Test dengan skip enabled
	service.skip = true
	valid, err := service.VerifyToken(context.Background(), "test-token")
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test dengan service disabled
	service.skip = false
	service.enabled = false
	valid, err = service.VerifyToken(context.Background(), "test-token")
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestRecaptchaService_VerifyToken_EmptyToken(t *testing.T) {
	log := logger.NewStructuredLogger(logger.LoggerConfig{Level: "debug", Format: "text", ServiceName: "recaptcha-test"})
	service := &recaptchaService{
		secretKey:     "test-secret-key",
		siteVerifyURL: "https://www.google.com/recaptcha/api/siteverify",
		timeout:       5 * time.Second,
		enabled:       true,
		skip:          false,
		logger:        log,
		httpClient:    &http.Client{},
	}

	valid, err := service.VerifyToken(context.Background(), "")
	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "reCAPTCHA token is required")
}

func TestRecaptchaService_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		skip     bool
		expected bool
	}{
		{
			name:     "enabled and not skip",
			enabled:  true,
			skip:     false,
			expected: true,
		},
		{
			name:     "enabled but skip",
			enabled:  true,
			skip:     true,
			expected: false,
		},
		{
			name:     "disabled",
			enabled:  false,
			skip:     false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &recaptchaService{
				enabled: tt.enabled,
				skip:    tt.skip,
			}
			assert.Equal(t, tt.expected, service.IsEnabled())
		})
	}
}

func TestRecaptchaResponse_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    RecaptchaResponse
		wantErr bool
	}{
		{
			name: "success response v2",
			json: `{
				"success": true,
				"challenge_ts": "2023-01-01T00:00:00Z",
				"hostname": "example.com"
			}`,
			want: RecaptchaResponse{
				Success:     true,
				ChallengeTS: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Hostname:    "example.com",
			},
			wantErr: false,
		},
		{
			name: "success response v3",
			json: `{
				"success": true,
				"score": 0.9,
				"action": "login",
				"challenge_ts": "2023-01-01T00:00:00Z",
				"hostname": "example.com"
			}`,
			want: RecaptchaResponse{
				Success:     true,
				Score:       0.9,
				Action:      "login",
				ChallengeTS: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Hostname:    "example.com",
			},
			wantErr: false,
		},
		{
			name: "failed response",
			json: `{
				"success": false,
				"error-codes": ["missing-input-secret", "invalid-input-secret"],
				"challenge_ts": "2023-01-01T00:00:00Z",
				"hostname": "example.com"
			}`,
			want: RecaptchaResponse{
				Success:     false,
				ErrorCodes:  []string{"missing-input-secret", "invalid-input-secret"},
				ChallengeTS: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Hostname:    "example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RecaptchaResponse
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
