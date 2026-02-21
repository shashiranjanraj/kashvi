package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
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

	var raw map[string]interface{}
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
		value := strings.TrimSpace(line[idx+1:])
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

func get(key, fallback string) string {
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
