package database

import (
	"fmt"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
	applogger "github.com/shashiranjanraj/kashvi/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect opens the database and configures the connection pool.
// Returns an error instead of calling log.Fatal so the caller can
// shut down gracefully. Safe to call multiple times; subsequent calls
// no-op if already connected (so auto-migrate can run after first connect).
func Connect() error {
	if DB != nil {
		return nil
	}
	driver := config.DatabaseDriver()
	dsn := config.DatabaseDSN()

	dialector, err := buildDialector(driver, dsn)
	if err != nil {
		return fmt.Errorf("database: build dialector: %w", err)
	}

	var gormLogLevel logger.LogLevel
	switch config.DatabaseLogMode() {
	case "info":
		gormLogLevel = logger.Info
	case "warn":
		gormLogLevel = logger.Warn
	case "error":
		gormLogLevel = logger.Error
	default:
		gormLogLevel = logger.Silent
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	}

	DB, err = gorm.Open(dialector, gormCfg)
	if err != nil {
		return fmt.Errorf("database: open: %w", err)
	}

	// Configure connection pool for production.
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("database: get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	// Verify connection is live.
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database: ping: %w", err)
	}

	applogger.Info("database connected", "driver", driver)

	return nil
}

func buildDialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "sqlite":
		return sqlite.Open(dsn), nil
	case "postgres":
		return postgres.Open(dsn), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "sqlserver":
		return sqlserver.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (supported: sqlite, postgres, mysql, sqlserver)", driver)
	}
}
