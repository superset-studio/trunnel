//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type connResponse struct {
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

func TestConnectionCRUD(t *testing.T) {
	truncateTables(t)

	// Register a user + org.
	auth := register(t, "conn@test.com", "password123", "Conn User", "Conn Org")
	client := newClient().withToken(auth.AccessToken)
	orgID := auth.Organization.ID

	// Create a connection.
	createBody := map[string]interface{}{
		"name":     "my-aws-conn",
		"category": "aws",
		"credentials": map[string]string{
			"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
			"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":          "us-east-1",
		},
		"config": map[string]string{
			"region": "us-east-1",
		},
	}
	rec := client.post("/api/v1/organizations/"+orgID+"/connections", createBody)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	created := decode[connResponse](t, rec)
	assert.Equal(t, "my-aws-conn", created.Name)
	assert.Equal(t, "aws", created.Category)
	assert.Equal(t, "pending", created.Status)
	assert.NotEmpty(t, created.ID)

	connID := created.ID

	// List connections.
	rec = client.get("/api/v1/organizations/" + orgID + "/connections")
	require.Equal(t, http.StatusOK, rec.Code)

	conns := decode[[]connResponse](t, rec)
	assert.Len(t, conns, 1)
	assert.Equal(t, connID, conns[0].ID)

	// Get connection by ID.
	rec = client.get("/api/v1/organizations/" + orgID + "/connections/" + connID)
	require.Equal(t, http.StatusOK, rec.Code)

	got := decode[connResponse](t, rec)
	assert.Equal(t, "my-aws-conn", got.Name)

	// Update connection name.
	updateBody := map[string]interface{}{
		"name": "renamed-conn",
	}
	rec = client.put("/api/v1/organizations/"+orgID+"/connections/"+connID, updateBody)
	require.Equal(t, http.StatusOK, rec.Code)

	updated := decode[connResponse](t, rec)
	assert.Equal(t, "renamed-conn", updated.Name)

	// Duplicate name conflict.
	createBody2 := map[string]interface{}{
		"name":     "second-conn",
		"category": "aws",
		"credentials": map[string]string{
			"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
			"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":          "us-west-2",
		},
	}
	rec = client.post("/api/v1/organizations/"+orgID+"/connections", createBody2)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Try to rename second to same as first.
	rec = client.put("/api/v1/organizations/"+orgID+"/connections/"+decode[connResponse](t, rec).ID, map[string]interface{}{
		"name": "renamed-conn",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	// Delete connection.
	rec = client.delete("/api/v1/organizations/" + orgID + "/connections/" + connID)
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Verify deleted.
	rec = client.get("/api/v1/organizations/" + orgID + "/connections/" + connID)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestConnectionTenantIsolation(t *testing.T) {
	truncateTables(t)

	// Register two users with different orgs.
	auth1 := register(t, "tenant1@test.com", "password123", "User 1", "Org 1")
	auth2 := register(t, "tenant2@test.com", "password123", "User 2", "Org 2")

	client1 := newClient().withToken(auth1.AccessToken)
	client2 := newClient().withToken(auth2.AccessToken)

	// Org 1 creates a connection.
	createBody := map[string]interface{}{
		"name":     "org1-conn",
		"category": "aws",
		"credentials": map[string]string{
			"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
			"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":          "us-east-1",
		},
	}
	rec := client1.post("/api/v1/organizations/"+auth1.Organization.ID+"/connections", createBody)
	require.Equal(t, http.StatusCreated, rec.Code)

	connID := decode[connResponse](t, rec).ID

	// Org 2 cannot see org 1's connection.
	rec = client2.get("/api/v1/organizations/" + auth2.Organization.ID + "/connections/" + connID)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Org 2's list is empty.
	rec = client2.get("/api/v1/organizations/" + auth2.Organization.ID + "/connections")
	require.Equal(t, http.StatusOK, rec.Code)

	conns := decode[[]connResponse](t, rec)
	assert.Len(t, conns, 0)
}
