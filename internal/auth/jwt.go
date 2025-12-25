package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents JWT token claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	refreshTokens   map[string]*RefreshToken // In-memory store; use Redis in production
}

// RefreshToken stores refresh token metadata
type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTTL, refreshTTL time.Duration) *JWTManager {
	if secretKey == "" {
		// Generate a random secret if none provided (not recommended for production)
		secretKey = generateRandomSecret()
	}

	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		refreshTokens:   make(map[string]*RefreshToken),
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(user *User) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := m.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := m.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.accessTokenTTL)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "wirescope",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// GenerateRefreshToken generates a new refresh token
func (m *JWTManager) GenerateRefreshToken(user *User) (string, error) {
	// Generate a random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tokenString := base64.URLEncoding.EncodeToString(b)

	// Store refresh token
	refreshToken := &RefreshToken{
		Token:     tokenString,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(m.refreshTokenTTL),
		CreatedAt: time.Now(),
	}
	m.refreshTokens[tokenString] = refreshToken

	return tokenString, nil
}

// ValidateAccessToken validates and parses an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*RefreshToken, error) {
	refreshToken, exists := m.refreshTokens[tokenString]
	if !exists {
		return nil, ErrInvalidToken
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		delete(m.refreshTokens, tokenString)
		return nil, ErrExpiredToken
	}

	return refreshToken, nil
}

// RevokeRefreshToken revokes a refresh token
func (m *JWTManager) RevokeRefreshToken(tokenString string) {
	delete(m.refreshTokens, tokenString)
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (m *JWTManager) RevokeAllUserTokens(userID string) {
	for token, rt := range m.refreshTokens {
		if rt.UserID == userID {
			delete(m.refreshTokens, token)
		}
	}
}

// CleanupExpiredTokens removes expired refresh tokens
func (m *JWTManager) CleanupExpiredTokens() {
	now := time.Now()
	for token, rt := range m.refreshTokens {
		if now.After(rt.ExpiresAt) {
			delete(m.refreshTokens, token)
		}
	}
}

// generateRandomSecret generates a random secret key
func generateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
