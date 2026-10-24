package crypt

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := Encrypt("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if enc == "hello world" {
		t.Fatal("ciphertext equals plaintext")
	}
	plain, err := Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "hello world" {
		t.Fatalf("Decrypt = %q", plain)
	}

	again, _ := Encrypt("hello world")
	if again == enc {
		t.Fatal("expected a fresh nonce per encryption")
	}
}

func TestDecryptRejectsTampering(t *testing.T) {
	enc, err := Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0x01
	if _, err := Decrypt(base64.URLEncoding.EncodeToString(raw)); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("tampered ciphertext: err = %v, want ErrDecrypt", err)
	}
	if _, err := Decrypt("not-base64!!"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestAppKeyIsUsed(t *testing.T) {
	t.Setenv("APP_KEY", "key-one")
	enc, err := Encrypt("data")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_KEY", "key-two")
	if _, err := Decrypt(enc); err == nil {
		t.Fatal("decrypting with a different APP_KEY should fail")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	enc, err := EncryptJSON(map[string]any{"user_id": 42.0})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := DecryptJSON(enc, &out); err != nil {
		t.Fatal(err)
	}
	if out["user_id"] != 42.0 {
		t.Fatalf("got %v", out)
	}
}
