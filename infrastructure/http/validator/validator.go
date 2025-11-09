package validator

import (
	"net/mail"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	
	// Gunakan net/mail untuk validasi email yang lebih komprehensif
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Tambahan validasi dengan regex untuk memastikan format yang lebih ketat
	return emailRegex.MatchString(strings.ToLower(email))
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	// Minimal 8 karakter, mengandung huruf besar, huruf kecil, angka, dan karakter khusus
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func ValidateRequired(value string) bool {
	return strings.TrimSpace(value) != ""
}

func ValidateJWT(token string) bool {
	if token == "" {
		return false
	}
	
	// JWT token harus memiliki 3 bagian yang dipisahkan oleh titik
	parts := strings.Split(token, ".")
	return len(parts) == 3
}