package entity

import (
	"time"
)

type User struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Password  string     `json:"-"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func NewUser(id, name, email, password, role, status string) *User {
	now := time.Now()
	return &User{
		ID:        id,
		Name:      name,
		Email:     email,
		Password:  password,
		Role:      role,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewUserWithDefaults creates a user with default values
func NewUserWithDefaults(id, name, email, password, role string) *User {
	return NewUser(id, name, email, password, role, "active")
}

// IsActive checks if user is active (not soft deleted and status is active)
func (u *User) IsActive() bool {
	return u.DeletedAt == nil && u.Status == "active"
}

// SoftDelete marks the user as deleted
func (u *User) SoftDelete() {
	now := time.Now()
	u.DeletedAt = &now
	u.UpdatedAt = now
}

// UpdateName updates user name
func (u *User) UpdateName(name string) {
	u.Name = name
	u.UpdatedAt = time.Now()
}

// UpdateRole updates user role
func (u *User) UpdateRole(role string) {
	u.Role = role
	u.UpdatedAt = time.Now()
}

// UpdateStatus updates user status
func (u *User) UpdateStatus(status string) {
	u.Status = status
	u.UpdatedAt = time.Now()
}