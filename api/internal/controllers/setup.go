package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	awsprovider "github.com/superset-studio/kapstan/api/internal/provider/aws"
)

// SetupController provides endpoints for platform setup helpers
// (e.g. detecting Kapstan's own cloud identity).
type SetupController struct{}

func NewSetupController() *SetupController { return &SetupController{} }

type awsIdentityResponse struct {
	AccountID string `json:"accountId"`
	ARN       string `json:"arn"`
}

// GetAWSIdentity returns the AWS account ID and ARN of the server's own
// ambient credentials so the frontend can build IAM trust policies.
func (ctrl *SetupController) GetAWSIdentity(c echo.Context) error {
	info, err := awsprovider.GetAmbientIdentity(c.Request().Context(), "")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "unable to detect server AWS identity")
	}
	return c.JSON(http.StatusOK, awsIdentityResponse{
		AccountID: info.AccountID,
		ARN:       info.UserARN,
	})
}
