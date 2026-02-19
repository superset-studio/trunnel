package repositories

import (
	"errors"
	"strings"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrDuplicateOrgName   = errors.New("organization name already exists")
	ErrDuplicateMembership = errors.New("user is already a member of this organization")
)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505")
}
