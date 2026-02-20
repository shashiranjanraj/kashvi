package database

import (
	"fmt"
	"log"

	"github.com/shashiranjanraj/kashvi/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	driver := config.DatabaseDriver()
	dsn := config.DatabaseDSN()

	dialector, err := buildDialector(driver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
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
