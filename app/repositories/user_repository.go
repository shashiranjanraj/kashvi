package repositories

import (
	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

// UserRepository handles database operations for User.
type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// FindByEmail looks up a user by their email address.
func (r *UserRepository) FindByEmail(email string) (models.User, error) {
	var user models.User
	err := orm.DB().Model(&models.User{}).Where("email = ?", email).First(&user)
	return user, err
}

// FindByID looks up a user by primary key.
func (r *UserRepository) FindByID(id uint) (models.User, error) {
	var user models.User
	err := orm.DB().Model(&models.User{}).Where("id = ?", id).First(&user)
	return user, err
}

// Create persists a new user record.
func (r *UserRepository) Create(user *models.User) error {
	return orm.DB().Create(user)
}

// Update persists changes to an existing user.
func (r *UserRepository) Update(user *models.User) error {
	return orm.DB().Save(user)
}

// All returns all users with optional pagination.
func (r *UserRepository) All(page, limit int) ([]models.User, orm.Pagination, error) {
	var users []models.User
	pagination, err := orm.DB().Model(&models.User{}).GetWithPagination(&users, page, limit)
	return users, pagination, err
}
