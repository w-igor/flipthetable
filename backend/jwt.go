package main

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// accessTokenTTL is the lifetime of short-lived access tokens (15 minutes)
	accessTokenTTL = 15 * time.Minute
	// refreshTokenTTL is the default lifetime of refresh tokens (24 hours)
	refreshTokenTTL = 24 * time.Hour
	// refreshTokenRememberTTL is the extended lifetime when "remember me" is checked (30 days)
	refreshTokenRememberTTL = 30 * 24 * time.Hour
)

// tokenClaims holds the custom JWT claims including user ID and token type.
type tokenClaims struct {
	UserID string `json:"uid"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// jwtSecret retrieves the JWT signing secret from environment or returns a dev default.
// WARNING: The default is for development only and is not secure for production.
func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-only-insecure-secret-change-me"
	}
	return []byte(secret)
}

// generateToken creates a signed JWT token for the specified user with given type and TTL.
// Returns the encoded token string or an error if signing fails.
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

// generateAuthTokens creates both access and refresh tokens for a user after login/register.
// If remember is true, the refresh token gets extended expiry (30 days instead of 24 hours).
func generateAuthTokens(userID string, remember bool) (accessToken string, refreshToken string, err error) {
	accessToken, err = generateToken(userID, "access", accessTokenTTL)
	if err != nil {
		return "", "", err
	}

	// Extend refresh token expiry if user wants to stay logged in
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

// parseToken validates and decodes a JWT token, ensuring it matches the expected token type.
// Returns the parsed claims or an error if the token is invalid, expired, or of wrong type.
func parseToken(tokenString string, expectedType string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// Ensure only HMAC signing method is accepted (prevent algorithm substitution attacks)
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
	// Verify the token type matches what the endpoint expects
	if claims.Type != expectedType {
		return nil, errors.New("unexpected token type")
	}
	return claims, nil
}
