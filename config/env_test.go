package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMergeDotEnvInlineComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := `# comment line
PLAIN=value
COMMENTED=2000           # max requests
EMPTY_COMMENTED=         # leave empty
TABBED=on	# tab comment
HASH_IN_VALUE=abc#def
QUOTED="has # hash"   # trailing
SINGLE='x y'
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	if err := mergeDotEnv(path, got); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"PLAIN":           "value",
		"COMMENTED":       "2000",
		"EMPTY_COMMENTED": "",
		"TABBED":          "on",
		"HASH_IN_VALUE":   "abc#def",
		"QUOTED":          "has # hash",
		"SINGLE":          "x y",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestOSEnvOverridesFiles(t *testing.T) {
	_ = Load()
	t.Setenv("JWT_SECRET", "from-env")
	if got := JWTSecret(); got != "from-env" {
		t.Fatalf("JWTSecret() = %q, want from-env", got)
	}
	t.Setenv("JWT_SECRET", "")
	if got := JWTSecret(); got != defaultJWTSecret {
		t.Fatalf("empty env var should fall through, got %q", got)
	}
}

func TestDurationsAndLists(t *testing.T) {
	_ = Load()
	if got := JWTTTL(); got != 24*time.Hour {
		t.Fatalf("default JWTTTL = %v", got)
	}
	t.Setenv("JWT_TTL", "15m")
	if got := JWTTTL(); got != 15*time.Minute {
		t.Fatalf("JWTTTL = %v, want 15m", got)
	}
	t.Setenv("JWT_REFRESH_TTL", "nonsense")
	if got := JWTRefreshTTL(); got != 7*24*time.Hour {
		t.Fatalf("invalid JWT_REFRESH_TTL should fall back, got %v", got)
	}

	if got := CORSAllowedOrigins(); !reflect.DeepEqual(got, []string{"*"}) {
		t.Fatalf("default CORSAllowedOrigins = %v", got)
	}
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example.com , ,https://b.example.com")
	if got := CORSAllowedOrigins(); !reflect.DeepEqual(got, []string{"https://a.example.com", "https://b.example.com"}) {
		t.Fatalf("CORSAllowedOrigins = %v", got)
	}
	if got := TrustedProxies(); got != nil {
		t.Fatalf("default TrustedProxies = %v, want nil", got)
	}
}

func TestIsProduction(t *testing.T) {
	_ = Load()
	for env, want := range map[string]bool{"production": true, "PROD": true, "local": false, "staging": false} {
		t.Setenv("APP_ENV", env)
		if got := IsProduction(); got != want {
			t.Errorf("APP_ENV=%s: IsProduction() = %v, want %v", env, got, want)
		}
	}
}
