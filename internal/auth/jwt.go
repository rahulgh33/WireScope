package auth
package auth

// JWT-based authentication for multi-tenant deployments
// Add organization ID to claims for tenant isolation

import (
	"context"
































































}	return orgID, ok	orgID, ok := ctx.Value(orgIDKey).(string)func GetOrgID(ctx context.Context) (string, bool) {}	return context.WithValue(ctx, orgIDKey, orgID)func WithOrgID(ctx context.Context, orgID string) context.Context {const orgIDKey contextKey = "org_id"type contextKey string}	return nil, errors.New("invalid token")	}		return claims, nil	if claims, ok := token.Claims.(*Claims); ok && token.Valid {	}		return nil, err	if err != nil {	})		return s.secret, nil	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {}	return token.SignedString(s.secret)	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)	}		},			IssuedAt:  jwt.NewNumericDate(time.Now()),			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),		RegisteredClaims: jwt.RegisteredClaims{		Role:   role,		OrgID:  orgID,		UserID: userID,	claims := Claims{func (s *AuthService) GenerateToken(userID, orgID, role string) (string, error) {}	return &AuthService{secret: []byte(secret)}func NewAuthService(secret string) *AuthService {}	secret []bytetype AuthService struct {}	jwt.RegisteredClaims	Role   string `json:"role"`	OrgID  string `json:"org_id"`	UserID string `json:"user_id"`type Claims struct {)	"github.com/golang-jwt/jwt/v5"	"time"	"errors"