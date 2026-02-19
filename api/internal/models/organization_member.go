package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

func RoleLevel(r Role) int {
	switch r {
	case RoleOwner:
		return 40
	case RoleAdmin:
		return 30
	case RoleMember:
		return 20
	case RoleViewer:
		return 10
	default:
		return 0
	}
}

func IsValidRole(s string) bool {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

type OrganizationMember struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	OrganizationID uuid.UUID  `db:"organization_id" json:"organizationId"`
	UserID         uuid.UUID  `db:"user_id" json:"userId"`
	Role           Role       `db:"role" json:"role"`
	InvitedBy      *uuid.UUID `db:"invited_by" json:"invitedBy,omitempty"`
	InvitedAt      *time.Time `db:"invited_at" json:"invitedAt,omitempty"`
	AcceptedAt     *time.Time `db:"accepted_at" json:"acceptedAt,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
}

type MemberWithUser struct {
	OrganizationMember
	Email     string  `db:"email" json:"email"`
	Name      string  `db:"name" json:"name"`
	AvatarURL *string `db:"avatar_url" json:"avatarUrl,omitempty"`
}
