package outbound

type TokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type TokenService interface {
	GenerateAccessToken(claims TokenClaims) (string, error)
	GenerateRefreshToken() (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
}