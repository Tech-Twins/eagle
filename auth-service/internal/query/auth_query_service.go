package query

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/eaglebank/auth-service/internal/repository"
	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/utils"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecretOnce sync.Once
	jwtSecretVal  []byte
)

func jwtSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			panic("JWT_SECRET environment variable is not set")
		}
		jwtSecretVal = []byte(secret)
	})
	return jwtSecretVal
}

// Claims is the JWT payload.
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthQueryService handles login and token refresh. There's no CommandService
// for auth because these operations don't mutate application state.
type AuthQueryService struct {
	userRepo *repository.UserRepository
}

func NewAuthQueryService(userRepo *repository.UserRepository) *AuthQueryService {
	return &AuthQueryService{userRepo: userRepo}
}

func (s *AuthQueryService) Login(cmd cqrs.LoginCommand) (string, error) {
	user, err := s.userRepo.GetByEmail(cmd.Email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	if !utils.CheckPassword(cmd.Password, user.PasswordHash) {
		return "", fmt.Errorf("invalid credentials")
	}
	return s.generateToken(user.ID, user.Email)
}

func (s *AuthQueryService) RefreshToken(cmd cqrs.RefreshTokenCommand) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cmd.Token, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return s.generateToken(claims.UserID, claims.Email)
}

func (s *AuthQueryService) generateToken(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret())
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return signed, nil
}
