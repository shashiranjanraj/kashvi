package database

import (
	"fmt"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
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
// shut down gracefully.
func Connect() error {
	driver := config.DatabaseDriver()
	dsn := config.DatabaseDSN()

	dialector, err := buildDialector(driver, dsn)
	if err != nil {
		return fmt.Errorf("database: build dialector: %w", err)
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // use pkg/logger, not GORM's own
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
