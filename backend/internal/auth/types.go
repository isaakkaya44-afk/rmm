package auth

import (
	"time"
)

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleTechnician Role = "technician"
	RoleCustomer   Role = "customer"
)

type User struct {
	ID             string    `json:"id" db:"id"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	FullName       string    `json:"full_name" db:"full_name"`
	RoleID         int       `json:"role_id" db:"role_id"`
	RoleName       string    `json:"role_name" db:"role_name"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         UserDTO `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	IsActive  bool   `json:"is_active"`
}

type TokenClaims struct {
	UserID   string `json:"uid"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}
