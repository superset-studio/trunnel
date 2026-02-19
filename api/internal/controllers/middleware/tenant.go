package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/models"
)

const (
	ContextKeyTenantID = "tenant_id"
	ContextKeyOrgRole  = "org_role"
)

func Tenant() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
			}

			orgIDStr := c.Param("org_id")
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid organization ID")
			}

			// Find matching org membership in JWT claims.
			var matchedRole models.Role
			found := false
			for _, m := range claims.OrgMemberships {
				if m.OrgID == orgID {
					matchedRole = m.Role
					found = true
					break
				}
			}

			if !found {
				return echo.NewHTTPError(http.StatusForbidden, "not a member of this organization")
			}

			c.Set(ContextKeyTenantID, orgID)
			c.Set(ContextKeyOrgRole, matchedRole)
			return next(c)
		}
	}
}

func GetTenantID(c echo.Context) uuid.UUID {
	id, _ := c.Get(ContextKeyTenantID).(uuid.UUID)
	return id
}

func GetOrgRole(c echo.Context) models.Role {
	role, _ := c.Get(ContextKeyOrgRole).(models.Role)
	return role
}
