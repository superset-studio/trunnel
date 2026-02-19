package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/models"
)

func RequireRole(minRole models.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := GetOrgRole(c)
			if models.RoleLevel(userRole) < models.RoleLevel(minRole) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}
