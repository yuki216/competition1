package entity

import (
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(id, email, password, role string) *User {
	now := time.Now()
	return &User{
		ID:        id,
		Email:     email,
		Password:  password,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}