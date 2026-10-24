package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shashiranjanraj/kashvi/config"
	"golang.org/x/crypto/bcrypt"
)

// TokenTypeRefresh marks a token issued by GenerateRefreshToken.
const TokenTypeRefresh = "refresh"

// ErrRefreshToken is returned by ValidateToken when given a refresh token.
var ErrRefreshToken = errors.New("auth: refresh token cannot be used as an access token")

// ErrNotRefreshToken is returned by ValidateRefreshToken when given an access token.
var ErrNotRefreshToken = errors.New("auth: token is not a refresh token")

// Claims holds the typed JWT payload.
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Type   string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

func secret() []byte {
	return []byte(config.JWTSecret())
}

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(userID uint, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

// GenerateRefreshToken creates a longer-lived token used to refresh access.
func GenerateRefreshToken(userID uint, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

// ValidateToken parses and validates an access token. Refresh tokens are rejected.
func ValidateToken(t string) (*Claims, error) {
	claims, err := parse(t)
	if err != nil {
		return nil, err
	}
	if claims.Type == TokenTypeRefresh {
		return nil, ErrRefreshToken
	}
	return claims, nil
}

// ValidateRefreshToken parses and validates a token issued by GenerateRefreshToken.
func ValidateRefreshToken(t string) (*Claims, error) {
	claims, err := parse(t)
	if err != nil {
		return nil, err
	}
	if claims.Type != TokenTypeRefresh {
		return nil, ErrNotRefreshToken
	}
	return claims, nil
}

func parse(t string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(t, &Claims{}, func(tok *jwt.Token) (any, error) {
		return secret(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// HashPassword returns a bcrypt hash of the plain-text password.
func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a bcrypt hash against the plain-text candidate.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
