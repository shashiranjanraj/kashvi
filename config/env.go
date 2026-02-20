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
		"DB_DRIVER":    defaultDatabaseDriver,
		"REDIS_ADDR":   defaultRedisAddr,
		"DATABASE_DSN": "",
	}
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
