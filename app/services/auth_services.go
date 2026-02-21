package services

import (
	"errors"

	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/auth"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// Login looks up the user by email, verifies the password and returns a signed JWT.
func (s *AuthService) Login(email, password string) (token string, refresh string, err error) {
	var user models.User

	if err = orm.DB().
		Model(&models.User{}).
		Where("email = ?", email).
		First(&user); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	if !auth.CheckPassword(user.Password, password) {
		return "", "", errors.New("invalid credentials")
	}

	token, err = auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", "", err
	}

	refresh, err = auth.GenerateRefreshToken(user.ID, user.Role)
	return token, refresh, err
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(name, email, password string) (models.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Name:     name,
		Email:    email,
		Password: hash,
		Role:     "user",
	}

	if err := orm.DB().Create(&user); err != nil {
		return models.User{}, err
	}

	return user, nil
}
