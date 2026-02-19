package controllers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/controllers/middleware"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

type APIKeyController struct {
	apiKeySvc *services.APIKeyService
}

func NewAPIKeyController(apiKeySvc *services.APIKeyService) *APIKeyController {
	return &APIKeyController{apiKeySvc: apiKeySvc}
}

type createAPIKeyRequest struct {
	Name        string `json:"name" validate:"required"`
	AccessLevel string `json:"accessLevel" validate:"required"`
}

type createAPIKeyResponse struct {
	APIKey interface{} `json:"apiKey"`
	Key    string      `json:"key"`
}

func (ctrl *APIKeyController) CreateAPIKey(c echo.Context) error {
	claims := middleware.GetClaims(c)
	tenantID := middleware.GetTenantID(c)

	var req createAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	key, rawKey, err := ctrl.apiKeySvc.CreateAPIKey(
		c.Request().Context(), tenantID, req.Name, req.AccessLevel, claims.UserID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create API key")
	}

	return c.JSON(http.StatusCreated, createAPIKeyResponse{
		APIKey: key,
		Key:    rawKey,
	})
}

func (ctrl *APIKeyController) ListAPIKeys(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)

	keys, err := ctrl.apiKeySvc.ListAPIKeys(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list API keys")
	}

	return c.JSON(http.StatusOK, keys)
}

func (ctrl *APIKeyController) RevokeAPIKey(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	keyID, err := uuid.Parse(c.Param("key_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid key ID")
	}

	if err := ctrl.apiKeySvc.RevokeAPIKey(c.Request().Context(), keyID, tenantID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "API key not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to revoke API key")
	}

	return c.NoContent(http.StatusNoContent)
}
