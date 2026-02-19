//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiTenantIsolation(t *testing.T) {
	truncateTables(t)

	// Register Alice (Alice Corp) and Bob (Bob Inc).
	alice := register(t, "alice@example.com", "password123", "Alice", "Alice Corp")
	bob := register(t, "bob@example.com", "password123", "Bob", "Bob Inc")

	aliceClient := newClient().withToken(alice.AccessToken)
	bobClient := newClient().withToken(bob.AccessToken)

	// Alice cannot GET Bob's org.
	rec := aliceClient.get(fmt.Sprintf("/api/v1/organizations/%s", bob.Organization.ID))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Bob cannot GET Alice's org.
	rec = bobClient.get(fmt.Sprintf("/api/v1/organizations/%s", alice.Organization.ID))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Alice cannot list Bob's org members.
	rec = aliceClient.get(fmt.Sprintf("/api/v1/organizations/%s/members", bob.Organization.ID))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Alice cannot invite into Bob's org.
	rec = aliceClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", bob.Organization.ID), map[string]string{
		"email": "intruder@example.com",
		"role":  "member",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Alice lists orgs — only her org.
	rec = aliceClient.get("/api/v1/organizations")
	require.Equal(t, http.StatusOK, rec.Code)
	var orgs []orgResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &orgs))
	require.Len(t, orgs, 1)
	assert.Equal(t, alice.Organization.ID, orgs[0].ID)
}

func TestRBACEnforcement(t *testing.T) {
	truncateTables(t)

	// Owner registers.
	owner := register(t, "owner@example.com", "password123", "Owner", "RBAC Corp")
	orgID := owner.Organization.ID
	ownerClient := newClient().withToken(owner.AccessToken)

	// Register the other users so they exist.
	register(t, "admin@example.com", "password123", "Admin User", "Admin Org")
	register(t, "member@example.com", "password123", "Member User", "Member Org")
	register(t, "viewer@example.com", "password123", "Viewer User", "Viewer Org")

	// Owner invites admin, member, viewer.
	rec := ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "admin@example.com",
		"role":  "admin",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "member@example.com",
		"role":  "member",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "viewer@example.com",
		"role":  "viewer",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	// Re-login each user to get tokens with updated memberships.
	adminAuth := login(t, "admin@example.com", "password123")
	memberAuth := login(t, "member@example.com", "password123")
	viewerAuth := login(t, "viewer@example.com", "password123")

	adminClient := newClient().withToken(adminAuth.AccessToken)
	memberClient := newClient().withToken(memberAuth.AccessToken)
	viewerClient := newClient().withToken(viewerAuth.AccessToken)

	// 1. All roles can GET org and GET members → 200.
	for name, c := range map[string]*testClient{
		"owner": ownerClient, "admin": adminClient, "member": memberClient, "viewer": viewerClient,
	} {
		rec = c.get(fmt.Sprintf("/api/v1/organizations/%s", orgID))
		assert.Equal(t, http.StatusOK, rec.Code, "%s GET org", name)

		rec = c.get(fmt.Sprintf("/api/v1/organizations/%s/members", orgID))
		assert.Equal(t, http.StatusOK, rec.Code, "%s GET members", name)
	}

	// 2. PUT org: owner/admin → 200, member/viewer → 403.
	updateBody := map[string]string{"displayName": "Updated Corp"}

	rec = ownerClient.put(fmt.Sprintf("/api/v1/organizations/%s", orgID), updateBody)
	assert.Equal(t, http.StatusOK, rec.Code, "owner PUT org")

	rec = adminClient.put(fmt.Sprintf("/api/v1/organizations/%s", orgID), updateBody)
	assert.Equal(t, http.StatusOK, rec.Code, "admin PUT org")

	rec = memberClient.put(fmt.Sprintf("/api/v1/organizations/%s", orgID), updateBody)
	assert.Equal(t, http.StatusForbidden, rec.Code, "member PUT org")

	rec = viewerClient.put(fmt.Sprintf("/api/v1/organizations/%s", orgID), updateBody)
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer PUT org")

	// 3. POST invite: owner/admin → 201, member/viewer → 403.
	// Register throwaway users for invite targets.
	register(t, "invite1@example.com", "password123", "Invite1", "Invite1 Org")
	register(t, "invite2@example.com", "password123", "Invite2", "Invite2 Org")

	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "invite1@example.com",
		"role":  "viewer",
	})
	assert.Equal(t, http.StatusCreated, rec.Code, "owner POST invite")

	rec = adminClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "invite2@example.com",
		"role":  "viewer",
	})
	assert.Equal(t, http.StatusCreated, rec.Code, "admin POST invite")

	rec = memberClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "shouldfail1@example.com",
		"role":  "viewer",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "member POST invite")

	rec = viewerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "shouldfail2@example.com",
		"role":  "viewer",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer POST invite")

	// 4. PUT member role: owner → 200, others → 403.
	// Get the admin member's ID from the members list.
	rec = ownerClient.get(fmt.Sprintf("/api/v1/organizations/%s/members", orgID))
	require.Equal(t, http.StatusOK, rec.Code)
	var members []memberResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &members))

	var adminMemberID string
	for _, m := range members {
		if m.Email == "admin@example.com" {
			adminMemberID = m.ID
			break
		}
	}
	require.NotEmpty(t, adminMemberID, "admin member not found")

	rec = adminClient.put(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, adminMemberID), map[string]string{
		"role": "admin",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "admin PUT member role")

	rec = memberClient.put(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, adminMemberID), map[string]string{
		"role": "admin",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "member PUT member role")

	rec = viewerClient.put(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, adminMemberID), map[string]string{
		"role": "admin",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer PUT member role")

	rec = ownerClient.put(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, adminMemberID), map[string]string{
		"role": "member",
	})
	assert.Equal(t, http.StatusOK, rec.Code, "owner PUT member role")

	// 5. DELETE member: owner/admin → 204, member/viewer → 403.
	// Find invite1's member ID (viewer, expendable).
	rec = ownerClient.get(fmt.Sprintf("/api/v1/organizations/%s/members", orgID))
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &members))

	var invite1MemberID, invite2MemberID string
	for _, m := range members {
		if m.Email == "invite1@example.com" {
			invite1MemberID = m.ID
		}
		if m.Email == "invite2@example.com" {
			invite2MemberID = m.ID
		}
	}
	require.NotEmpty(t, invite1MemberID, "invite1 member not found")
	require.NotEmpty(t, invite2MemberID, "invite2 member not found")

	// member/viewer cannot delete.
	rec = memberClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, invite1MemberID))
	assert.Equal(t, http.StatusForbidden, rec.Code, "member DELETE member")

	rec = viewerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, invite1MemberID))
	assert.Equal(t, http.StatusForbidden, rec.Code, "viewer DELETE member")

	// owner can delete.
	rec = ownerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, invite1MemberID))
	assert.Equal(t, http.StatusNoContent, rec.Code, "owner DELETE member")

	// Re-login admin (role was changed to member above, change back to admin for delete test).
	rec = ownerClient.put(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, adminMemberID), map[string]string{
		"role": "admin",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	adminAuth = login(t, "admin@example.com", "password123")
	adminClient = newClient().withToken(adminAuth.AccessToken)

	rec = adminClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, invite2MemberID))
	assert.Equal(t, http.StatusNoContent, rec.Code, "admin DELETE member")
}

func TestMemberLifecycle(t *testing.T) {
	truncateTables(t)

	// 1. Owner registers.
	owner := register(t, "owner@example.com", "password123", "Owner", "Lifecycle Corp")
	orgID := owner.Organization.ID
	ownerClient := newClient().withToken(owner.AccessToken)

	// 2. Bob registers separately (has his own org).
	register(t, "bob@example.com", "password123", "Bob", "Bob Inc")

	// 3. Owner invites Bob → 201.
	rec := ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "bob@example.com",
		"role":  "member",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var invitedMember memberResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &invitedMember))
	bobMemberID := invitedMember.ID

	// 4. Bob re-logins, GET org → 200, list orgs → 2.
	bobAuth := login(t, "bob@example.com", "password123")
	bobClient := newClient().withToken(bobAuth.AccessToken)

	rec = bobClient.get(fmt.Sprintf("/api/v1/organizations/%s", orgID))
	assert.Equal(t, http.StatusOK, rec.Code, "bob GET org after invite")

	rec = bobClient.get("/api/v1/organizations")
	require.Equal(t, http.StatusOK, rec.Code)
	var orgs []orgResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &orgs))
	assert.Len(t, orgs, 2, "bob should see 2 orgs")

	// 5. Owner removes Bob → 204.
	rec = ownerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, bobMemberID))
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// 6. Bob re-logins, GET org → 403, list orgs → 1.
	bobAuth = login(t, "bob@example.com", "password123")
	bobClient = newClient().withToken(bobAuth.AccessToken)

	rec = bobClient.get(fmt.Sprintf("/api/v1/organizations/%s", orgID))
	assert.Equal(t, http.StatusForbidden, rec.Code, "bob GET org after removal")

	rec = bobClient.get("/api/v1/organizations")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &orgs))
	assert.Len(t, orgs, 1, "bob should see 1 org after removal")

	// 7. Re-invite → 201.
	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "bob@example.com",
		"role":  "member",
	})
	assert.Equal(t, http.StatusCreated, rec.Code, "re-invite bob")

	// 8. Duplicate invite → 409.
	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "bob@example.com",
		"role":  "member",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, "duplicate invite")

	// 9. Remove last owner → 400.
	// Find owner's own member ID.
	rec = ownerClient.get(fmt.Sprintf("/api/v1/organizations/%s/members", orgID))
	require.Equal(t, http.StatusOK, rec.Code)
	var members []memberResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &members))

	var ownerMemberID string
	for _, m := range members {
		if m.Email == "owner@example.com" {
			ownerMemberID = m.ID
			break
		}
	}
	require.NotEmpty(t, ownerMemberID)

	rec = ownerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, ownerMemberID))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "cannot remove last owner")

	errResp := decode[errorBody](t, rec)
	assert.Contains(t, errResp.Message, "cannot remove the last owner")
}

func TestAPIKeyLifecycle(t *testing.T) {
	truncateTables(t)

	// Owner registers.
	owner := register(t, "owner@example.com", "password123", "Owner", "APIKey Corp")
	orgID := owner.Organization.ID
	ownerClient := newClient().withToken(owner.AccessToken)

	// 1. Owner creates key → 201, raw key in response.
	rec := ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID), map[string]string{
		"name":        "Production Key",
		"accessLevel": "admin",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.Key, "raw key should be returned")
	assert.NotEmpty(t, created.APIKey.ID)
	assert.Equal(t, "Production Key", created.APIKey.Name)
	assert.Equal(t, "admin", created.APIKey.AccessLevel)
	ownerKeyID := created.APIKey.ID

	// 2. List → key visible without raw value.
	rec = ownerClient.get(fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID))
	require.Equal(t, http.StatusOK, rec.Code)
	var keys []apiKeyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &keys))
	require.Len(t, keys, 1)
	assert.Equal(t, "Production Key", keys[0].Name)

	// 3. Invite a member and have them create a key → 201 (any role can create).
	register(t, "member@example.com", "password123", "Member", "Member Org")
	rec = ownerClient.post(fmt.Sprintf("/api/v1/organizations/%s/members/invite", orgID), map[string]string{
		"email": "member@example.com",
		"role":  "member",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	memberAuth := login(t, "member@example.com", "password123")
	memberClient := newClient().withToken(memberAuth.AccessToken)

	rec = memberClient.post(fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID), map[string]string{
		"name":        "Member Key",
		"accessLevel": "read",
	})
	assert.Equal(t, http.StatusCreated, rec.Code, "member create key")

	var memberKeyCreated apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &memberKeyCreated))
	memberKeyID := memberKeyCreated.APIKey.ID

	// 4. Member revoke → 403 (needs admin).
	rec = memberClient.delete(fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", orgID, ownerKeyID))
	assert.Equal(t, http.StatusForbidden, rec.Code, "member revoke key")

	// 5. Owner revokes owner's key → 204.
	rec = ownerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", orgID, ownerKeyID))
	assert.Equal(t, http.StatusNoContent, rec.Code, "owner revoke key")

	// 6. List → revoked key gone (only member's key remains).
	rec = ownerClient.get(fmt.Sprintf("/api/v1/organizations/%s/api-keys", orgID))
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &keys))
	assert.Len(t, keys, 1)
	assert.Equal(t, memberKeyID, keys[0].ID)

	// 7. Revoke again → 404.
	rec = ownerClient.delete(fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", orgID, ownerKeyID))
	assert.Equal(t, http.StatusNotFound, rec.Code, "revoke already-revoked key")
}
