package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/platform/auth"
)

const ContextKeyClaims = "claims"

func JWTAuth(jwtManager *auth.JWTManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			claims, err := jwtManager.ValidateAccessToken(parts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(ContextKeyClaims, claims)
			return next(c)
		}
	}
}

func GetClaims(c echo.Context) *models.Claims {
	claims, ok := c.Get(ContextKeyClaims).(*models.Claims)
	if !ok {
		return nil
	}
	return claims
}
