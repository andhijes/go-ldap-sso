package auth

import (
	"fmt"
	"go-ldap-sso/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateToken(email string, scopes []string, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"sub":    email,
		"scopes": scopes,
		"exp":    time.Now().Add(time.Duration(cfg.AuthConfig.JWTExpiryHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.AuthConfig.JWTSecret))
}

func ValidateToken(tokenString string, cfg *config.Config) (email string, scopes []string, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Pastikan metode signing cocok
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.AuthConfig.JWTSecret), nil
	})

	if err != nil {
		return "", nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", nil, fmt.Errorf("invalid token claims")
	}

	// Ambil email
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", nil, fmt.Errorf("missing subject (email)")
	}

	// Ambil scopes
	rawScopes, ok := claims["scopes"].([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("invalid scopes")
	}

	// Convert interface{} slice to []string
	var scopesStr []string
	for _, s := range rawScopes {
		if str, ok := s.(string); ok {
			scopesStr = append(scopesStr, str)
		}
	}

	return sub, scopesStr, nil
}
