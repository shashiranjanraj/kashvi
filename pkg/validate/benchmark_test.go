package validate_test

import (
	"testing"

	"github.com/shashiranjanraj/kashvi/pkg/validate"
)

// BenchmarkStructValid measures validation of a valid struct (all rules pass).
func BenchmarkStructValid(b *testing.B) {
	input := signupInput{
		Name:                 "john_doe",
		Email:                "john@example.com",
		Password:             "secret123",
		PasswordConfirmation: "secret123",
		Age:                  25,
		Role:                 "user",
		Website:              "",
		DeviceIP:             "192.168.1.1",
		Score:                85.5,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validate.Struct(input)
	}
}

// BenchmarkStructInvalid measures validation when required fields are missing (early exit).
func BenchmarkStructInvalid(b *testing.B) {
	input := signupInput{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validate.Struct(input)
	}
}
