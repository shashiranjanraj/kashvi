package auth

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestAccessToken(t *testing.T) {
	tok, err := GenerateToken(7, "admin")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if c.UserID != 7 || c.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", c)
	}
	if _, err := ValidateRefreshToken(tok); !errors.Is(err, ErrNotRefreshToken) {
		t.Fatalf("ValidateRefreshToken(access) err = %v, want ErrNotRefreshToken", err)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	tok, err := GenerateRefreshToken(7, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(tok); !errors.Is(err, ErrRefreshToken) {
		t.Fatalf("ValidateToken(refresh) err = %v, want ErrRefreshToken", err)
	}
	c, err := ValidateRefreshToken(tok)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if c.UserID != 7 {
		t.Fatalf("unexpected claims: %+v", c)
	}
}

func TestRejectsOtherSigningMethods(t *testing.T) {
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{UserID: 1}).SignedString(secret())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(tok); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}
}

func TestPassword(t *testing.T) {
	h, err := HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "s3cret") || CheckPassword(h, "wrong") {
		t.Fatal("CheckPassword mismatch")
	}
}
