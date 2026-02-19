package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/services"
)

func APIKeyAuth(apiKeySvc *services.APIKeyService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing API key")
			}

			key, err := apiKeySvc.ValidateAPIKey(c.Request().Context(), apiKey)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
			}

			c.Set(ContextKeyTenantID, key.TenantID)
			return next(c)
		}
	}
}
