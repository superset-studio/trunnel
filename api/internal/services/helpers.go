package services

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = nonAlphaNum.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func hashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
