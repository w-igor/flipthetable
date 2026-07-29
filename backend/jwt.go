package main

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL         = 15 * time.Minute
	refreshTokenTTL        = 24 * time.Hour
	refreshTokenRememberTTL = 30 * 24 * time.Hour
)

type tokenClaims struct {
	UserID string `json:"uid"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-only-insecure-secret-change-me"
	}
	return []byte(secret)
}

func generateToken(userID string, tokenType string, ttl time.Duration) (string, error) {
	claims := tokenClaims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func generateAuthTokens(userID string, remember bool) (accessToken string, refreshToken string, err error) {
	accessToken, err = generateToken(userID, "access", accessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshTTL := refreshTokenTTL
	if remember {
		refreshTTL = refreshTokenRememberTTL
	}
	refreshToken, err = generateToken(userID, "refresh", refreshTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func parseToken(tokenString string, expectedType string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != expectedType {
		return nil, errors.New("unexpected token type")
	}
	return claims, nil
}
