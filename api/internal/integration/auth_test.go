//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRegisterAndLogin(t *testing.T) {
	truncateTables(t)

	// 1. Register Alice — creates user + org.
	alice := register(t, "alice@example.com", "password123", "Alice Smith", "Alice Corp")

	assert.NotEmpty(t, alice.AccessToken)
	assert.NotEmpty(t, alice.RefreshToken)
	assert.Equal(t, "alice@example.com", alice.User.Email)
	assert.Equal(t, "Alice Smith", alice.User.Name)
	assert.Equal(t, "Alice Corp", alice.Organization.DisplayName)
	assert.Equal(t, "alice-corp", alice.Organization.Name) // slug
	assert.NotEmpty(t, alice.User.ID)
	assert.NotEmpty(t, alice.Organization.ID)

	// 2. Token works — GET /organizations returns her org.
	c := newClient().withToken(alice.AccessToken)
	rec := c.get("/api/v1/organizations")
	require.Equal(t, http.StatusOK, rec.Code)

	var orgs []orgResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &orgs))
	require.Len(t, orgs, 1)
	assert.Equal(t, alice.Organization.ID, orgs[0].ID)

	// 3. Login same credentials — returns same org.
	aliceLogin := login(t, "alice@example.com", "password123")
	assert.Equal(t, alice.Organization.ID, aliceLogin.Organization.ID)
	assert.Equal(t, alice.User.ID, aliceLogin.User.ID)

	// 4. Refresh — returns new tokens.
	rec = newClient().post("/api/v1/auth/refresh", map[string]string{
		"refreshToken": aliceLogin.RefreshToken,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	var refreshed tokenPairResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &refreshed))
	assert.NotEmpty(t, refreshed.AccessToken)
	assert.NotEmpty(t, refreshed.RefreshToken)
	// Refresh token must differ (rotation). Access token may be identical if
	// issued within the same second with the same claims.
	assert.NotEqual(t, aliceLogin.RefreshToken, refreshed.RefreshToken)

	// 5. Reuse old refresh token — should fail (rotation).
	rec = newClient().post("/api/v1/auth/refresh", map[string]string{
		"refreshToken": aliceLogin.RefreshToken,
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthRejections(t *testing.T) {
	truncateTables(t)

	// Seed: register alice so we can test duplicates.
	register(t, "alice@example.com", "password123", "Alice Smith", "Alice Corp")

	// 1. Duplicate email → 409.
	rec := newClient().post("/api/v1/auth/register", map[string]string{
		"email":    "alice@example.com",
		"password": "different123",
		"name":     "Another Alice",
		"orgName":  "Another Corp",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	// 2. Duplicate org name → 409.
	rec = newClient().post("/api/v1/auth/register", map[string]string{
		"email":    "bob@example.com",
		"password": "password123",
		"name":     "Bob",
		"orgName":  "Alice Corp", // same slug
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	// 3. Login non-existent email → 401.
	rec = newClient().post("/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "password123",
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 4. Login wrong password → 401.
	rec = newClient().post("/api/v1/auth/login", map[string]string{
		"email":    "alice@example.com",
		"password": "wrongpassword",
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 5. Register missing fields → 400.
	rec = newClient().post("/api/v1/auth/register", map[string]string{
		"email": "partial@example.com",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// 6. No token on protected endpoint → 401.
	rec = newClient().get("/api/v1/organizations")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 7. Garbage token → 401.
	rec = newClient().withToken("not.a.valid.jwt").get("/api/v1/organizations")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
