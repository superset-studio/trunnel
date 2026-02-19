package controllers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (ctrl *AuthController) Register(c echo.Context) error {
	var req models.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := ctrl.authService.Register(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			return echo.NewHTTPError(http.StatusConflict, "email already exists")
		}
		if errors.Is(err, repositories.ErrDuplicateOrgName) {
			return echo.NewHTTPError(http.StatusConflict, "organization name already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "registration failed")
	}

	return c.JSON(http.StatusCreated, resp)
}

func (ctrl *AuthController) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	resp, err := ctrl.authService.Login(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	return c.JSON(http.StatusOK, resp)
}

func (ctrl *AuthController) Refresh(c echo.Context) error {
	var req models.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	tokenPair, err := ctrl.authService.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "token refresh failed")
	}

	return c.JSON(http.StatusOK, tokenPair)
}
