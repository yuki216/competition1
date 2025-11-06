package valueobject

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

type Credentials struct {
	email    string
	password string
}

func NewCredentials(email, password string) (*Credentials, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	return &Credentials{
		email:    email,
		password: password,
	}, nil
}

func (c *Credentials) Email() string {
	return c.email
}

func (c *Credentials) Password() string {
	return c.password
}

func validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}