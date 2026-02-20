package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/shashiranjanraj/kashvi/app/services"
)

type AuthController struct {
	service *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		service: services.NewAuthService(),
	}
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string
	}

	json.NewDecoder(r.Body).Decode(&body)

	token, err := c.service.Login(body.Email)
	if err != nil {
		http.Error(w, "Invalid user", 401)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
