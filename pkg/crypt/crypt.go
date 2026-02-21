// Package crypt provides AES-GCM authenticated encryption helpers for Kashvi.
//
// All ciphertext is base64url-encoded and includes the random nonce prefix,
// so a single string can be safely stored in a DB column or cookie.
//
// Usage:
//
//	enc, err := crypt.Encrypt("hello world")
//	plain, err := crypt.Decrypt(enc)
//
//	// Typed helpers
//	enc, _ := crypt.EncryptJSON(map[string]any{"user_id": 42})
//	var out map[string]any
//	crypt.DecryptJSON(enc, &out)
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/shashiranjanraj/kashvi/config"
)

// ErrDecrypt is returned when decryption or authentication fails.
var ErrDecrypt = errors.New("crypt: decryption failed")

// key derives a 32-byte AES-256 key from the APP_KEY / JWT_SECRET config value.
func key() ([]byte, error) {
	secret := config.Get("APP_KEY", config.JWTSecret())
	if secret == "" {
		return nil, errors.New("crypt: APP_KEY not configured")
	}
	// Always derive a fixed-length key via SHA-256.
	h := sha256.Sum256([]byte(secret))
	return h[:], nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64url string.
// The output format is: base64url(nonce || ciphertext || tag)
func Encrypt(plaintext string) (string, error) {
	return EncryptBytes([]byte(plaintext))
}

// EncryptBytes encrypts raw bytes and returns a base64url string.
func EncryptBytes(data []byte) (string, error) {
	k, err := key()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(k)
	if err != nil {
		return "", fmt.Errorf("crypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypt: new GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypt: nonce: %w", err)
	}

	// Seal appends ciphertext+tag after nonce.
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64url string produced by Encrypt.
func Decrypt(encoded string) (string, error) {
	b, err := DecryptBytes(encoded)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecryptBytes decrypts a base64url string and returns raw bytes.
func DecryptBytes(encoded string) ([]byte, error) {
	k, err := key()
	if err != nil {
		return nil, err
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrDecrypt
	}

	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("crypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypt: new GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrDecrypt
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plain, nil
}

// EncryptJSON marshals v to JSON then encrypts it.
func EncryptJSON(v interface{}) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("crypt: marshal: %w", err)
	}
	return EncryptBytes(raw)
}

// DecryptJSON decrypts encoded and unmarshals the result into dest.
func DecryptJSON(encoded string, dest interface{}) error {
	raw, err := DecryptBytes(encoded)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("crypt: unmarshal: %w", err)
	}
	return nil
}

// Hash returns a SHA-256 hex digest of the input — useful for checksums.
func Hash(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h)
}
