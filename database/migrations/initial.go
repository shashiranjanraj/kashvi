package migrations

import (
	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/migration"
	"gorm.io/gorm"
)

func init() {
	migration.Register("20260101000000_create_users_table", &CreateUsersTable{})
	migration.Register("20260101000001_create_orders_table", &CreateOrdersTable{})
}

// -------- 0001: users --------

type CreateUsersTable struct{}

func (m *CreateUsersTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{})
}

func (m *CreateUsersTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("users")
}

// -------- 0002: orders --------

type CreateOrdersTable struct{}

func (m *CreateOrdersTable) Up(db *gorm.DB) error {
	return db.AutoMigrate(&models.Order{})
}

func (m *CreateOrdersTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("orders")
}
