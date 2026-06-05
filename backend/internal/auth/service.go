package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rmm-platform/backend/internal/shared/config"
)

type Service struct {
	repo   *Repository
	config *config.JWTConfig
}

func NewService(repo *Repository, cfg *config.JWTConfig) *Service {
	return &Service{repo: repo, config: cfg}
}

func (s *Service) Login(email, password string) (*LoginResponse, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !s.repo.VerifyPassword(user.PasswordHash, password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("refresh token generation failed: %w", err)
	}

	if err := s.repo.UpdateLastLogin(user.ID); err != nil {
		return nil, fmt.Errorf("failed to update login: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.AccessTokenTTL.Seconds()),
		User: UserDTO{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     user.RoleName,
			IsActive: user.IsActive,
		},
	}, nil
}

func (s *Service) Refresh(refreshToken string) (*LoginResponse, error) {
	tokenHash := hashToken(refreshToken)

	userID, err := s.repo.ValidateRefreshToken(tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	user, err := s.repo.FindByID(*userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if err := s.repo.DeleteRefreshToken(tokenHash); err != nil {
		return nil, fmt.Errorf("failed to delete old token")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("refresh token generation failed: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.AccessTokenTTL.Seconds()),
		User: UserDTO{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     user.RoleName,
			IsActive: user.IsActive,
		},
	}, nil
}

func (s *Service) Logout(refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.repo.DeleteRefreshToken(tokenHash)
}

func (s *Service) GetUser(userID string) (*UserDTO, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return &UserDTO{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     user.RoleName,
		IsActive: user.IsActive,
	}, nil
}

func (s *Service) generateAccessToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"uid":   user.ID,
		"email": user.Email,
		"role":  user.RoleName,
		"exp":   time.Now().Add(s.config.AccessTokenTTL).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Secret))
}

func (s *Service) generateRefreshToken(userID string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	tokenHash := hashToken(token)

	if err := s.repo.SaveRefreshToken(userID, tokenHash, time.Now().Add(s.config.RefreshTokenTTL)); err != nil {
		return "", err
	}

	return token, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
