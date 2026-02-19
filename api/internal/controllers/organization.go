package controllers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/controllers/middleware"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

type OrganizationController struct {
	orgService *services.OrganizationService
}

func NewOrganizationController(orgService *services.OrganizationService) *OrganizationController {
	return &OrganizationController{orgService: orgService}
}

type createOrgRequest struct {
	DisplayName string `json:"displayName" validate:"required"`
}

type updateOrgRequest struct {
	DisplayName string  `json:"displayName" validate:"required"`
	LogoURL     *string `json:"logoUrl"`
}

func (ctrl *OrganizationController) CreateOrganization(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
	}

	var req createOrgRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	org, err := ctrl.orgService.CreateOrganization(c.Request().Context(), claims.UserID, req.DisplayName)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateOrgName) {
			return echo.NewHTTPError(http.StatusConflict, "organization name already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create organization")
	}

	return c.JSON(http.StatusCreated, org)
}

func (ctrl *OrganizationController) ListOrganizations(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
	}

	orgs, err := ctrl.orgService.ListUserOrganizations(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list organizations")
	}

	return c.JSON(http.StatusOK, orgs)
}

func (ctrl *OrganizationController) GetOrganization(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)

	org, err := ctrl.orgService.GetOrganization(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "organization not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get organization")
	}

	return c.JSON(http.StatusOK, org)
}

func (ctrl *OrganizationController) UpdateOrganization(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)

	var req updateOrgRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	org, err := ctrl.orgService.GetOrganization(c.Request().Context(), tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "organization not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get organization")
	}

	org.DisplayName = req.DisplayName
	org.LogoURL = req.LogoURL

	if err := ctrl.orgService.UpdateOrganization(c.Request().Context(), org); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update organization")
	}

	return c.JSON(http.StatusOK, org)
}
