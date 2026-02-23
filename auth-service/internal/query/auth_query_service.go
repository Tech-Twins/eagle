package query

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eaglebank/auth-service/internal/repository"
	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/utils"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecretVal []byte

// MustInitJWTSecret reads JWT_SECRET from the environment and stores it for
// use by the auth query service. It must be called once at service startup
// before any requests are served. The process exits immediately if the
// variable is unset so the misconfiguration is caught at boot time.
func MustInitJWTSecret() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}
	jwtSecretVal = []byte(secret)
}

func jwtSecret() []byte {
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
