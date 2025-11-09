package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type BcryptPasswordService struct {
	cost int
}

func NewBcryptPasswordService(cost int) *BcryptPasswordService {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptPasswordService{
		cost: cost,
	}
}

func (s *BcryptPasswordService) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedPassword), nil
}

func (s *BcryptPasswordService) ComparePassword(hashedPassword, password string) error {
	if hashedPassword == "" || password == "" {
		return fmt.Errorf("passwords cannot be empty")
	}

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (s *BcryptPasswordService) VerifyPassword(password, hash string) (bool, error) {
	if hash == "" || password == "" {
		return false, fmt.Errorf("passwords cannot be empty")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("failed to compare passwords: %w", err)
	}

	return true, nil
}