package validate_test

import (
	"testing"

	"github.com/shashiranjanraj/kashvi/pkg/validate"
)

type signupInput struct {
	Name                 string  `json:"name"                  validate:"required,alpha_dash,min=2,max=50"`
	Email                string  `json:"email"                 validate:"required,email"`
	Password             string  `json:"password"              validate:"required,min=8"`
	PasswordConfirmation string  `json:"password_confirmation" validate:"confirmed"`
	Age                  int     `json:"age"                   validate:"required,gte=18,lte=120"`
	Role                 string  `json:"role"                  validate:"required,in=admin,user,moderator"`
	Website              string  `json:"website"               validate:"nullable,url"`
	DeviceIP             string  `json:"device_ip"             validate:"required,ip"`
	Score                float64 `json:"score"                 validate:"required,between=0,100"`
}

func TestValidInput(t *testing.T) {
	errs := validate.Struct(signupInput{
		Name:                 "john_doe",
		Email:                "john@example.com",
		Password:             "secret123",
		PasswordConfirmation: "secret123",
		Age:                  25,
		Role:                 "user",
		Website:              "", // nullable — allowed to be empty
		DeviceIP:             "192.168.1.1",
		Score:                85.5,
	})
	if validate.HasErrors(errs) {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestRequiredFails(t *testing.T) {
	errs := validate.Struct(signupInput{})
	if !validate.HasErrors(errs) {
		t.Error("expected required errors")
	}
	if _, ok := errs["name"]; !ok {
		t.Error("expected name to be required")
	}
	if _, ok := errs["email"]; !ok {
		t.Error("expected email to be required")
	}
}

func TestEmailRule(t *testing.T) {
	type in struct {
		Email string `json:"email" validate:"required,email"`
	}
	errs := validate.Struct(in{Email: "not-an-email"})
	if _, ok := errs["email"]; !ok {
		t.Error("expected email validation error")
	}
	errs = validate.Struct(in{Email: "valid@example.com"})
	if validate.HasErrors(errs) {
		t.Errorf("expected valid email to pass, got: %v", errs)
	}
}

func TestNumericBounds(t *testing.T) {
	type in struct {
		Age int `json:"age" validate:"required,gte=18,lte=120"`
	}
	if errs := validate.Struct(in{Age: 15}); !validate.HasErrors(errs) {
		t.Error("expected age < 18 to fail")
	}
	if errs := validate.Struct(in{Age: 25}); validate.HasErrors(errs) {
		t.Errorf("expected age 25 to pass, got: %v", errs)
	}
}

func TestInRule(t *testing.T) {
	type in struct {
		Role string `json:"role" validate:"required,in=admin,user,moderator"`
	}
	if errs := validate.Struct(in{Role: "superadmin"}); !validate.HasErrors(errs) {
		t.Error("expected invalid role to fail")
	}
	if errs := validate.Struct(in{Role: "admin"}); validate.HasErrors(errs) {
		t.Errorf("expected admin to pass: %v", errs)
	}
}

func TestConfirmedRule(t *testing.T) {
	type in struct {
		Password             string `json:"password"              validate:"required,min=8"`
		PasswordConfirmation string `json:"password_confirmation" validate:"confirmed"`
	}
	if errs := validate.Struct(in{Password: "secret123", PasswordConfirmation: "wrong"}); !validate.HasErrors(errs) {
		t.Error("expected confirmation mismatch to fail")
	}
	if errs := validate.Struct(in{Password: "secret123", PasswordConfirmation: "secret123"}); validate.HasErrors(errs) {
		t.Errorf("expected matching confirmation to pass: %v", errs)
	}
}

func TestNullableSkipsRules(t *testing.T) {
	type in struct {
		Website string `json:"website" validate:"nullable,url"`
	}
	// Empty string — nullable, should pass even though it's not a URL
	if errs := validate.Struct(in{Website: ""}); validate.HasErrors(errs) {
		t.Errorf("expected empty nullable to pass: %v", errs)
	}
	// Non-empty but invalid URL — should fail
	if errs := validate.Struct(in{Website: "not-a-url"}); !validate.HasErrors(errs) {
		t.Error("expected invalid URL to fail")
	}
}

func TestBetweenRule(t *testing.T) {
	type in struct {
		Score float64 `json:"score" validate:"required,between=0,100"`
	}
	if errs := validate.Struct(in{Score: 150}); !validate.HasErrors(errs) {
		t.Error("expected score > 100 to fail")
	}
	if errs := validate.Struct(in{Score: 75}); validate.HasErrors(errs) {
		t.Errorf("expected score 75 to pass: %v", errs)
	}
}

func TestURLRule(t *testing.T) {
	type in struct {
		Site string `json:"site" validate:"required,url"`
	}
	if errs := validate.Struct(in{Site: "https://kashvi.dev"}); validate.HasErrors(errs) {
		t.Errorf("expected valid URL to pass: %v", errs)
	}
	if errs := validate.Struct(in{Site: "not-a-url"}); !validate.HasErrors(errs) {
		t.Error("expected invalid URL to fail")
	}
}

func TestIPRule(t *testing.T) {
	type in struct {
		IP string `json:"ip" validate:"required,ip"`
	}
	if errs := validate.Struct(in{IP: "192.168.0.1"}); validate.HasErrors(errs) {
		t.Errorf("expected valid IP to pass: %v", errs)
	}
	if errs := validate.Struct(in{IP: "999.999.0.1"}); !validate.HasErrors(errs) {
		t.Error("expected invalid IP to fail")
	}
}

func TestAlphaDashRule(t *testing.T) {
	type in struct {
		Slug string `json:"slug" validate:"required,alpha_dash"`
	}
	if errs := validate.Struct(in{Slug: "hello-world_123"}); validate.HasErrors(errs) {
		t.Errorf("expected alpha_dash to pass: %v", errs)
	}
	if errs := validate.Struct(in{Slug: "hello world!"}); !validate.HasErrors(errs) {
		t.Error("expected alpha_dash to fail for spaces/punctuation")
	}
}
