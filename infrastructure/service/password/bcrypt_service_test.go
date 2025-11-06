package password

import (
	"testing"
)

func TestBcryptPasswordService(t *testing.T) {
	service := NewBcryptPasswordService(10)

	t.Run("HashPassword", func(t *testing.T) {
		password := "test-password-123"
		hash, err := service.HashPassword(password)
		if err != nil {
			t.Errorf("Failed to hash password: %v", err)
		}
		if hash == "" {
			t.Error("Hash should not be empty")
		}
	})

	t.Run("HashEmptyPassword", func(t *testing.T) {
		_, err := service.HashPassword("")
		if err == nil {
			t.Error("Should fail to hash empty password")
		}
	})

	t.Run("VerifyPassword", func(t *testing.T) {
		password := "test-password-123"
		hash, err := service.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		isValid, err := service.VerifyPassword(password, hash)
		if err != nil {
			t.Errorf("Failed to verify password: %v", err)
		}
		if !isValid {
			t.Error("Password should be valid")
		}
	})

	t.Run("VerifyWrongPassword", func(t *testing.T) {
		password := "test-password-123"
		wrongPassword := "wrong-password-456"
		hash, err := service.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		isValid, err := service.VerifyPassword(wrongPassword, hash)
		if err != nil {
			t.Errorf("Should not return error for wrong password: %v", err)
		}
		if isValid {
			t.Error("Wrong password should not be valid")
		}
	})

	t.Run("VerifyEmptyPassword", func(t *testing.T) {
		hash := "$2a$10$SomeValidHashString1234567890123456789012345678901234567890"
		_, err := service.VerifyPassword("", hash)
		if err == nil {
			t.Error("Should fail to verify with empty password")
		}
	})

	t.Run("VerifyEmptyHash", func(t *testing.T) {
		_, err := service.VerifyPassword("password", "")
		if err == nil {
			t.Error("Should fail to verify with empty hash")
		}
	})
}