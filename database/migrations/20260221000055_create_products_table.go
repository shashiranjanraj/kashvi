package migrations

import (
	"github.com/shashiranjanraj/kashvi/pkg/migration"
	"gorm.io/gorm"
)

func init() {
	migration.Register("20260221000055_create_products_table", &M_20260221000055_create_products_table{})
}

type M_20260221000055_create_products_table struct{}

func (m *M_20260221000055_create_products_table) Up(db *gorm.DB) error {
	// TODO: implement
	return nil
}

func (m *M_20260221000055_create_products_table) Down(db *gorm.DB) error {
	// TODO: implement
	return nil
}
