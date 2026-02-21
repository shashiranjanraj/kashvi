package models

import "gorm.io/gorm"

// Product represents a product in the catalogue.
type Product struct {
	gorm.Model
	Name        string  `gorm:"size:255;not null;index" json:"name"`
	Description string  `gorm:"type:text"              json:"description"`
	Price       float64 `gorm:"not null;default:0"     json:"price"`
	Stock       int     `gorm:"not null;default:0"     json:"stock"`
	SKU         string  `gorm:"size:100;uniqueIndex"   json:"sku"`
}
