package models

import "gorm.io/gorm"

// User is the primary user model.
type User struct {
	gorm.Model
	Name     string `gorm:"size:255;not null" json:"name"`
	Email    string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password string `gorm:"size:255;not null" json:"-"` // hashed, never serialised
	Role     string `gorm:"size:50;default:user" json:"role"`
}

// Order is a simple order model linked to a user.
type Order struct {
	gorm.Model
	UserID uint    `gorm:"not null;index" json:"user_id"`
	Total  float64 `json:"total"`
	Status string  `gorm:"size:50;default:pending" json:"status"`
}
