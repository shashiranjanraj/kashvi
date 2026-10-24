package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultDatabaseDriver = "sqlite"
	defaultSQLiteDSN      = "kashvi.db"
	defaultPostgresDSN    = "host=localhost user=postgres password=postgres dbname=kashvi port=5432 sslmode=disable"
	defaultMySQLDSN       = "root:root@tcp(127.0.0.1:3306)/kashvi?charset=utf8mb4&parseTime=True&loc=Local"
	defaultSQLServerDSN   = "sqlserver://sa:Your_password123@localhost:1433?database=kashvi"
	defaultRedisAddr      = "localhost:6379"
	defaultJWTSecret      = "change-me-in-production"
	defaultAppPort        = "8080"
	defaultAppEnv         = "local"
	defaultLogLevel       = "info"
	defaultDBLogMode      = "silent"
)

var (
	loadOnce sync.Once
	loadErr  error

	mu     sync.RWMutex
	values = defaultValues()
)

func Load() error {
	loadOnce.Do(func() {
		loadErr = loadFromFiles("config/app.json", ".env")
	})
	return loadErr
}

func DatabaseDriver() string {
	_ = Load()

	driver := strings.ToLower(get("DB_DRIVER", defaultDatabaseDriver))
	switch driver {
	case "sqlite", "postgres", "mysql", "sqlserver":
		return driver
	default:
		return defaultDatabaseDriver
	}
}

func DatabaseDSN() string {
	_ = Load()

	override := get("DATABASE_DSN", "")
	if override != "" {
		return override
	}

	switch DatabaseDriver() {
	case "postgres":
		return defaultPostgresDSN
	case "mysql":
		return defaultMySQLDSN
	case "sqlserver":
		return defaultSQLServerDSN
	default:
		return defaultSQLiteDSN
	}
}

func RedisAddr() string {
	_ = Load()
	return get("REDIS_ADDR", defaultRedisAddr)
}

func defaultValues() map[string]string {
	return map[string]string{
		"DB_DRIVER":      defaultDatabaseDriver,
		"REDIS_ADDR":     defaultRedisAddr,
		"DATABASE_DSN":   "",
		"JWT_SECRET":     defaultJWTSecret,
		"APP_PORT":       defaultAppPort,
		"APP_ENV":        defaultAppEnv,
		"REDIS_PASSWORD": "",
		"LOG_LEVEL":      defaultLogLevel,
		"DB_LOG_MODE":    defaultDBLogMode,
	}
}

func JWTSecret() string {
	_ = Load()
	return get("JWT_SECRET", defaultJWTSecret)
}

func AppPort() string {
	_ = Load()
	return get("APP_PORT", defaultAppPort)
}

func AppEnv() string {
	_ = Load()
	return get("APP_ENV", defaultAppEnv)
}

func LogLevel() string {
	_ = Load()
	return get("LOG_LEVEL", defaultLogLevel)
}

func DatabaseLogMode() string {
	_ = Load()
	return get("DB_LOG_MODE", defaultDBLogMode)
}

func RedisPassword() string {
	_ = Load()
	return get("REDIS_PASSWORD", "")
}

// ── Storage ──────────────────────────────────────────────────────────────────

func StorageDefault() string {
	_ = Load()
	return get("STORAGE_DISK", "local")
}

func StorageLocalRoot() string {
	_ = Load()
	return get("STORAGE_LOCAL_ROOT", "storage")
}

func StorageURL() string {
	_ = Load()
	return get("STORAGE_URL", "http://localhost:8080/storage")
}

func StorageS3Bucket() string   { _ = Load(); return get("S3_BUCKET", "") }
func StorageS3Region() string   { _ = Load(); return get("S3_REGION", "us-east-1") }
func StorageS3Key() string      { _ = Load(); return get("S3_KEY", "") }
func StorageS3Secret() string   { _ = Load(); return get("S3_SECRET", "") }
func StorageS3Endpoint() string { _ = Load(); return get("S3_ENDPOINT", "") }
func StorageS3URL() string      { _ = Load(); return get("S3_URL", "") }

// ── MongoDB ───────────────────────────────────────────────────────────────────

// MongoURI returns the MongoDB connection string (empty = disabled).
func MongoURI() string { _ = Load(); return get("MONGO_URI", "") }

// MongoLogDB returns the database name used for application logs.
func MongoLogDB() string { _ = Load(); return get("MONGO_LOG_DB", "kashvi_logs") }

// MongoLogCollection returns the collection name used for application logs.
func MongoLogCollection() string { _ = Load(); return get("MONGO_LOG_COLLECTION", "app_logs") }

// ── gRPC ──────────────────────────────────────────────────────────────────────

// GRPCPort returns the port the gRPC server listens on.
func GRPCPort() string { _ = Load(); return get("GRPC_PORT", "9090") }

// GRPCEnabled returns true if the gRPC server should be started.
func GRPCEnabled() bool { _ = Load(); return get("GRPC_ENABLED", "false") == "true" }

// ── Concurrency ───────────────────────────────────────────────────────────────

// WorkerPoolSize returns the bounded goroutine pool size.
func WorkerPoolSize() int {
	_ = Load()
	v := get("WORKER_POOL_SIZE", "50")
	n := 50
	fmt.Sscanf(v, "%d", &n) //nolint:errcheck
	if n <= 0 {
		n = 50
	}
	return n
}

// RateLimitMax returns the maximum number of requests per minute per IP.
func RateLimitMax() int {
	_ = Load()
	v := get("RATE_LIMIT_MAX", "2000")
	n := 2000
	fmt.Sscanf(v, "%d", &n) //nolint:errcheck
	if n <= 0 {
		n = 2000
	}
	return n
}

// ── Auth & HTTP ───────────────────────────────────────────────────────────────

// JWTTTL returns the access-token lifetime (JWT_TTL, default 24h).
func JWTTTL() time.Duration { _ = Load(); return duration("JWT_TTL", 24*time.Hour) }

// JWTRefreshTTL returns the refresh-token lifetime (JWT_REFRESH_TTL, default 168h).
func JWTRefreshTTL() time.Duration { _ = Load(); return duration("JWT_REFRESH_TTL", 7*24*time.Hour) }

// JWTIssuer returns the expected "iss" claim (JWT_ISSUER). Empty disables the check.
func JWTIssuer() string { _ = Load(); return get("JWT_ISSUER", "") }

// JWTAudience returns the expected "aud" claim (JWT_AUDIENCE). Empty disables the check.
func JWTAudience() string { _ = Load(); return get("JWT_AUDIENCE", "") }

// AppKey returns the encryption key used by pkg/crypt (APP_KEY). Empty means unset.
func AppKey() string { _ = Load(); return get("APP_KEY", "") }

// IsProduction reports whether APP_ENV is "production" or "prod".
func IsProduction() bool {
	env := strings.ToLower(AppEnv())
	return env == "production" || env == "prod"
}

// CORSAllowedOrigins returns the comma-separated CORS_ALLOWED_ORIGINS list (default "*").
func CORSAllowedOrigins() []string { _ = Load(); return list("CORS_ALLOWED_ORIGINS", "*") }

// TrustedProxies returns the comma-separated TRUSTED_PROXIES list of IPs or CIDRs
// whose X-Forwarded-For header is trusted. Empty means the middleware default.
func TrustedProxies() []string { _ = Load(); return list("TRUSTED_PROXIES", "") }

func duration(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(get(key, ""))
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func list(key, fallback string) []string {
	var out []string
	for _, item := range strings.Split(get(key, fallback), ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func loadFromFiles(configPath, envPath string) error {
	loaded := defaultValues()

	if err := mergeJSONConfig(configPath, loaded); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	if err := mergeDotEnv(envPath, loaded); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	mu.Lock()
	values = loaded
	mu.Unlock()

	return nil
}

func mergeJSONConfig(path string, out map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var raw map[string]any
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	for key, val := range raw {
		s, ok := val.(string)
		if !ok {
			continue
		}

		k := strings.ToUpper(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(s)
	}

	return nil
}

func mergeDotEnv(path string, out map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}

		key := strings.ToUpper(strings.TrimSpace(line[:idx]))
		value := stripInlineComment(strings.TrimSpace(line[idx+1:]))
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		out[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

// stripInlineComment removes a trailing " # comment" from an unquoted value.
func stripInlineComment(value string) string {
	if value == "" {
		return value
	}
	if q := value[0]; q == '"' || q == '\'' {
		// Quoted: keep everything up to the closing quote, drop what follows.
		if j := strings.IndexByte(value[1:], q); j >= 0 {
			return value[:j+2]
		}
		return value
	}
	if value[0] == '#' {
		return ""
	}
	if i := strings.Index(value, " #"); i >= 0 {
		value = value[:i]
	}
	if i := strings.Index(value, "\t#"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

// get resolves a key with precedence: OS environment > .env > config/app.json > fallback.
func get(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	mu.RLock()
	defer mu.RUnlock()

	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}

	return fallback
}

// Get reads any config key by name with an optional fallback.
// Keys from .env and app.json are available after config.Load().
func Get(key, fallback string) string {
	_ = Load()
	return get(key, fallback)
}
