package controllers

import (
	"net/http"

	"github.com/shashiranjanraj/kashvi/app/repositories"
	"github.com/shashiranjanraj/kashvi/app/services"
	"github.com/shashiranjanraj/kashvi/pkg/bind"
	"github.com/shashiranjanraj/kashvi/pkg/middleware"
	"github.com/shashiranjanraj/kashvi/pkg/response"
)

// AuthController handles authentication endpoints.
type AuthController struct {
	service *services.AuthService
	users   *repositories.UserRepository
}

func NewAuthController() *AuthController {
	return &AuthController{
		service: services.NewAuthService(),
		users:   repositories.NewUserRepository(),
	}
}

// ----- Register -----

type registerRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// Register POST /api/register
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	errs, err := bind.JSON(r, &req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	user, err := c.service.Register(req.Name, req.Email, req.Password)
	if err != nil {
		response.Error(w, http.StatusConflict, "Email already in use")
		return
	}

	response.Created(w, user)
}

// ----- Login -----

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Login POST /api/login
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	errs, err := bind.JSON(r, &req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	token, refresh, err := c.service.Login(req.Email, req.Password)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	response.Success(w, map[string]string{
		"token":         token,
		"refresh_token": refresh,
	})
}

// ----- Profile -----

// Profile GET /api/profile
func (c *AuthController) Profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r)
	if !ok {
		response.Unauthorized(w)
		return
	}

	user, err := c.users.FindByID(userID)
	if err != nil {
		response.NotFound(w)
		return
	}

	response.Success(w, user)
}

// ----- Update Profile -----

type updateProfileRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

// UpdateProfile PUT /api/profile
func (c *AuthController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r)
	if !ok {
		response.Unauthorized(w)
		return
	}

	var req updateProfileRequest
	errs, err := bind.JSON(r, &req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(errs) > 0 {
		response.ValidationError(w, errs)
		return
	}

	user, err := c.users.FindByID(userID)
	if err != nil {
		response.NotFound(w)
		return
	}

	user.Name = req.Name
	if err := c.users.Update(&user); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	response.Success(w, user)
}
