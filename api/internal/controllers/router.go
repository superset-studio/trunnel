package controllers

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/superset-studio/kapstan/api/internal/controllers/middleware"
	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/platform/auth"
	"github.com/superset-studio/kapstan/api/internal/platform/validate"
	"github.com/superset-studio/kapstan/api/internal/repositories"
	"github.com/superset-studio/kapstan/api/internal/services"
)

func NewRouter(db *sqlx.DB, jwtSecret []byte) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Validator = validate.NewValidator()
	e.HTTPErrorHandler = CustomHTTPErrorHandler

	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3456"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-Key"},
	}))

	// Health checks (no auth).
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/readyz", readyzHandler(db))

	// Construct dependencies.
	jwtManager := auth.NewJWTManager(jwtSecret)

	userRepo := repositories.NewUserRepository(db)
	orgRepo := repositories.NewOrganizationRepository(db)
	memberRepo := repositories.NewOrganizationMemberRepository(db)
	apiKeyRepo := repositories.NewAPIKeyRepository(db)
	tokenRepo := repositories.NewRefreshTokenRepository(db)

	authService := services.NewAuthService(db, jwtManager, userRepo, orgRepo, memberRepo, tokenRepo)
	orgService := services.NewOrganizationService(db, orgRepo, memberRepo, userRepo)
	apiKeySvc := services.NewAPIKeyService(apiKeyRepo)

	authCtrl := NewAuthController(authService)
	orgCtrl := NewOrganizationController(orgService)
	memberCtrl := NewMemberController(orgService)
	apiKeyCtrl := NewAPIKeyController(apiKeySvc)

	// Public auth routes.
	authGroup := e.Group("/api/v1/auth")
	authGroup.POST("/register", authCtrl.Register)
	authGroup.POST("/login", authCtrl.Login)
	authGroup.POST("/refresh", authCtrl.Refresh)

	// JWT-protected routes.
	jwtMiddleware := middleware.JWTAuth(jwtManager)

	orgsGroup := e.Group("/api/v1/organizations", jwtMiddleware)
	orgsGroup.POST("", orgCtrl.CreateOrganization)
	orgsGroup.GET("", orgCtrl.ListOrganizations)

	// Tenant-scoped routes (JWT + Tenant middleware).
	tenantGroup := orgsGroup.Group("/:org_id", middleware.Tenant())
	tenantGroup.GET("", orgCtrl.GetOrganization)
	tenantGroup.PUT("", orgCtrl.UpdateOrganization, middleware.RequireRole(models.RoleAdmin))

	// Members.
	membersGroup := tenantGroup.Group("/members")
	membersGroup.GET("", memberCtrl.ListMembers)
	membersGroup.POST("/invite", memberCtrl.InviteMember, middleware.RequireRole(models.RoleAdmin))
	membersGroup.PUT("/:member_id", memberCtrl.UpdateMemberRole, middleware.RequireRole(models.RoleOwner))
	membersGroup.DELETE("/:member_id", memberCtrl.RemoveMember, middleware.RequireRole(models.RoleAdmin))

	// API Keys.
	apiKeysGroup := tenantGroup.Group("/api-keys")
	apiKeysGroup.POST("", apiKeyCtrl.CreateAPIKey)
	apiKeysGroup.GET("", apiKeyCtrl.ListAPIKeys)
	apiKeysGroup.DELETE("/:key_id", apiKeyCtrl.RevokeAPIKey, middleware.RequireRole(models.RoleAdmin))

	return e
}

func readyzHandler(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"error":  "database unreachable",
			})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
