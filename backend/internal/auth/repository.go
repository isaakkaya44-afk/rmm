package auth

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByEmail(email string) (*User, error) {
	var user User
	query := `SELECT u.*, r.name as role_name FROM users u 
		JOIN roles r ON r.id = u.role_id 
		WHERE u.email = $1 AND u.is_active = TRUE`
	err := r.db.Get(&user, query, email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

func (r *Repository) FindByID(id string) (*User, error) {
	var user User
	query := `SELECT u.*, r.name as role_name FROM users u 
		JOIN roles r ON r.id = u.role_id 
		WHERE u.id = $1`
	err := r.db.Get(&user, query, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

func (r *Repository) VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (r *Repository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *Repository) SaveRefreshToken(userID, tokenHash string, expiresAt interface{}) error {
	_, err := r.db.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) DeleteRefreshToken(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *Repository) DeleteUserRefreshTokens(userID string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) ValidateRefreshToken(tokenHash string) (*string, error) {
	var userID string
	err := r.db.Get(&userID,
		`SELECT user_id FROM refresh_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()`, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}
	return &userID, nil
}
