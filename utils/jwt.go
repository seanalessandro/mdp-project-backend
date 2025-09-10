package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func getJwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return []byte("default-secret-for-dev-only")
	}
	return []byte(secret)
}

func getRefreshSecret() []byte {
	secret := os.Getenv("JWT_REFRESH_SECRET_KEY")
	if secret == "" {
		return []byte("default-refresh-secret-for-dev-only")
	}
	return []byte(secret)
}

// GenerateJWT generates an access token with shorter expiration
func GenerateJWT(username, role, userID string) (string, error) {
	claims := &Claims{
		Username: username,
		Role:     role,
		UserID:   userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Short-lived access token
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJwtSecret())
}

// GenerateRefreshToken generates a refresh token with longer expiration
func GenerateRefreshToken(username, userID string) (string, error) {
	claims := &RefreshClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getRefreshSecret())
}

func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getRefreshSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid refresh token")
	}
	return claims, nil
}
