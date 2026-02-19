package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/superset-studio/kapstan/api/internal/controllers/middleware"
	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

type ConnectionController struct {
	connService *services.ConnectionService
}

func NewConnectionController(connService *services.ConnectionService) *ConnectionController {
	return &ConnectionController{connService: connService}
}

type createConnectionRequest struct {
	Name        string                   `json:"name" validate:"required"`
	Category    models.ConnectionCategory `json:"category" validate:"required"`
	Credentials json.RawMessage          `json:"credentials" validate:"required"`
	Config      json.RawMessage          `json:"config"`
}

type updateConnectionRequest struct {
	Name        *string         `json:"name"`
	Credentials json.RawMessage `json:"credentials"`
	Config      json.RawMessage `json:"config"`
}

type connectionResponse struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenantId"`
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Status        string          `json:"status"`
	LastValidated *string         `json:"lastValidated,omitempty"`
	Config        json.RawMessage `json:"config,omitempty"`
	CreatedBy     *string         `json:"createdBy,omitempty"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
}

func connectionToResponse(conn *models.Connection) connectionResponse {
	resp := connectionResponse{
		ID:       conn.ID.String(),
		TenantID: conn.TenantID.String(),
		Name:     conn.Name,
		Category: string(conn.Category),
		Status:   string(conn.Status),
		Config:   conn.Config,
		CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if conn.LastValidated != nil {
		s := conn.LastValidated.Format("2006-01-02T15:04:05Z07:00")
		resp.LastValidated = &s
	}
	if conn.CreatedBy != nil {
		s := conn.CreatedBy.String()
		resp.CreatedBy = &s
	}
	return resp
}

func (ctrl *ConnectionController) CreateConnection(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
	}

	var req createConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if req.Category != models.ConnectionCategoryAWS {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported category; only 'aws' is supported")
	}

	conn, err := ctrl.connService.CreateConnection(
		c.Request().Context(), tenantID, req.Name, req.Category, req.Credentials, req.Config, claims.UserID,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateConnectionName) {
			return echo.NewHTTPError(http.StatusConflict, "connection name already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create connection")
	}

	return c.JSON(http.StatusCreated, connectionToResponse(conn))
}

func (ctrl *ConnectionController) ListConnections(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)

	conns, err := ctrl.connService.ListConnections(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list connections")
	}

	resp := make([]connectionResponse, len(conns))
	for i := range conns {
		resp[i] = connectionToResponse(&conns[i])
	}

	return c.JSON(http.StatusOK, resp)
}

func (ctrl *ConnectionController) GetConnection(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	connID, err := uuid.Parse(c.Param("conn_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection ID")
	}

	conn, err := ctrl.connService.GetConnection(c.Request().Context(), connID, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "connection not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get connection")
	}

	return c.JSON(http.StatusOK, connectionToResponse(conn))
}

func (ctrl *ConnectionController) UpdateConnection(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	connID, err := uuid.Parse(c.Param("conn_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection ID")
	}

	var req updateConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	conn, err := ctrl.connService.UpdateConnection(
		c.Request().Context(), connID, tenantID, req.Name, req.Credentials, req.Config,
	)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "connection not found")
		}
		if errors.Is(err, repositories.ErrDuplicateConnectionName) {
			return echo.NewHTTPError(http.StatusConflict, "connection name already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update connection")
	}

	return c.JSON(http.StatusOK, connectionToResponse(conn))
}

func (ctrl *ConnectionController) DeleteConnection(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	connID, err := uuid.Parse(c.Param("conn_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection ID")
	}

	if err := ctrl.connService.DeleteConnection(c.Request().Context(), connID, tenantID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "connection not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete connection")
	}

	return c.NoContent(http.StatusNoContent)
}

func (ctrl *ConnectionController) ValidateConnection(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	connID, err := uuid.Parse(c.Param("conn_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection ID")
	}

	conn, err := ctrl.connService.ValidateConnection(c.Request().Context(), connID, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "connection not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate connection")
	}

	return c.JSON(http.StatusOK, connectionToResponse(conn))
}
