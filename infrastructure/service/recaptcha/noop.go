package recaptcha

import (
	"context"

	"github.com/fixora/fixora/infrastructure/service/logger"
)

// noopRecaptchaService implements RecaptchaService as a no-op.
type noopRecaptchaService struct {
	logger logger.Logger
}

// NewNoopRecaptchaService returns a RecaptchaService that is always disabled
// and treats any token as valid without external verification.
func NewNoopRecaptchaService(log logger.Logger) RecaptchaService {
	return &noopRecaptchaService{logger: log}
}

func (n *noopRecaptchaService) VerifyToken(ctx context.Context, token string) (bool, error) {
	if n.logger != nil {
		// Keep log minimal to avoid noise in tests and disabled mode
		n.logger.Debug(ctx, "noop reCAPTCHA: verification skipped", map[string]interface{}{})
	}
	return true, nil
}

func (n *noopRecaptchaService) IsEnabled() bool {
	return false
}
