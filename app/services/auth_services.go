package services

import (
	"github.com/shashiranjanraj/kashvi/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/auth"
	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Login(email string) (string, error) {
	var user models.User

	err := orm.DB().
		Model(&models.User{}).
		Where("email = ?", email).
		First(&user)

	if err != nil {
		return "", err
	}

	return auth.GenerateToken(user.ID)
}
