package models

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
	OrgName  string `json:"orgName" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string        `json:"accessToken"`
	RefreshToken string        `json:"refreshToken"`
	User         *User         `json:"user"`
	Organization *Organization `json:"organization"`
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type OrgMembership struct {
	OrgID uuid.UUID `json:"orgId"`
	Role  Role      `json:"role"`
}

type Claims struct {
	jwt.RegisteredClaims
	UserID         uuid.UUID       `json:"userId"`
	Email          string          `json:"email"`
	OrgMemberships []OrgMembership `json:"orgMemberships"`
}
