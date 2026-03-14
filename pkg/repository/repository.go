// Package repository provides a repository layer for data access in Kashvi.
//
// Controllers and services should depend on repositories instead of calling
// orm.DB() directly. This keeps data access in one place and makes testing
// and swapping implementations easier.
//
// Use the generic Base[T] for standard CRUD, or embed it in a custom repository
// to add domain-specific queries.
//
//	// In app/repositories/user.go
//	type UserRepository struct {
//	    repository.Base[models.User]
//	}
//	func NewUserRepository() *UserRepository { return &UserRepository{} }
//
//	// In controller: repo.FindByID(id), repo.All(), repo.Create(&user), etc.
package repository

import (
	"github.com/shashiranjanraj/kashvi/pkg/orm"
	"gorm.io/gorm"
)

// Base provides generic CRUD operations for a model type T.
// Embed it in your concrete repository (e.g. UserRepository) so that
// controllers and services use the repository instead of orm directly.
//
// T should be a struct that works with GORM (e.g. has gorm.Model or
// an ID field and table name).
type Base[T any] struct{}

// Query returns a new orm.Query scoped to the model T.
// Use this for custom queries (filters, joins, pagination) while keeping
// data access inside the repository.
//
//	func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
//	    var u models.User
//	    err := r.Query().Where("email = ?", email).First(&u)
//	    if err != nil { return nil, err }
//	    return &u, nil
//	}
func (b *Base[T]) Query() *orm.Query {
	return orm.DB().Model(new(T))
}

// FindByID loads one record by primary key. Returns (nil, gorm.ErrRecordNotFound)
// when not found.
func (b *Base[T]) FindByID(id uint) (*T, error) {
	var dest T
	err := orm.DB().Model(new(T)).Where("id = ?", id).First(&dest)
	if err != nil {
		return nil, err
	}
	return &dest, nil
}

// All returns all records for the model (respecting soft deletes if T has DeletedAt).
func (b *Base[T]) All() ([]T, error) {
	var list []T
	err := orm.DB().Model(new(T)).Get(&list)
	return list, err
}

// Create inserts the model. The passed pointer is updated with generated fields (e.g. ID).
func (b *Base[T]) Create(m *T) error {
	return orm.DB().Create(m)
}

// Update saves the model (by primary key). Use after loading and modifying the struct.
func (b *Base[T]) Update(m *T) error {
	return orm.DB().Save(m)
}

// Delete soft-deletes (or hard-deletes if T has no DeletedAt) the record with the given ID.
func (b *Base[T]) Delete(id uint) error {
	return orm.DB().Delete(new(T), id)
}

// Exists returns true if a record with the given ID exists.
func (b *Base[T]) Exists(id uint) (bool, error) {
	_, err := b.FindByID(id)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Count returns the total number of records for the model (respecting soft deletes).
func (b *Base[T]) Count() (int64, error) {
	var count int64
	err := orm.DB().Model(new(T)).Count(&count)
	return count, err
}

// Paginate returns a slice of T and pagination metadata for the given page and limit.
func (b *Base[T]) Paginate(page, limit int) ([]T, orm.Pagination, error) {
	var list []T
	pag, err := orm.DB().Model(new(T)).GetWithPagination(&list, page, limit)
	return list, pag, err
}
