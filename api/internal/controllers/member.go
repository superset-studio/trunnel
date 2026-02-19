package controllers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/controllers/middleware"
	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

type MemberController struct {
	orgService *services.OrganizationService
}

func NewMemberController(orgService *services.OrganizationService) *MemberController {
	return &MemberController{orgService: orgService}
}

type inviteMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required"`
}

type updateRoleRequest struct {
	Role string `json:"role" validate:"required"`
}

func (ctrl *MemberController) ListMembers(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)

	members, err := ctrl.orgService.ListMembers(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list members")
	}

	return c.JSON(http.StatusOK, members)
}

func (ctrl *MemberController) InviteMember(c echo.Context) error {
	claims := middleware.GetClaims(c)
	tenantID := middleware.GetTenantID(c)

	var req inviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if !models.IsValidRole(req.Role) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	member, err := ctrl.orgService.InviteMember(
		c.Request().Context(), tenantID, claims.UserID, req.Email, models.Role(req.Role),
	)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateMembership) {
			return echo.NewHTTPError(http.StatusConflict, "user is already a member")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to invite member")
	}

	return c.JSON(http.StatusCreated, member)
}

func (ctrl *MemberController) UpdateMemberRole(c echo.Context) error {
	memberID, err := uuid.Parse(c.Param("member_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member ID")
	}

	var req updateRoleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if !models.IsValidRole(req.Role) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	if err := ctrl.orgService.UpdateMemberRole(c.Request().Context(), memberID, models.Role(req.Role)); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update member role")
	}

	return c.NoContent(http.StatusOK)
}

func (ctrl *MemberController) RemoveMember(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	memberID, err := uuid.Parse(c.Param("member_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member ID")
	}

	if err := ctrl.orgService.RemoveMember(c.Request().Context(), memberID, tenantID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		if errors.Is(err, services.ErrCannotRemoveLastOwner) {
			return echo.NewHTTPError(http.StatusBadRequest, "cannot remove the last owner")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove member")
	}

	return c.NoContent(http.StatusNoContent)
}
