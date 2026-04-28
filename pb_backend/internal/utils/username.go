package utils

import (
	"regexp"
	"strings"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	vkCollision     = regexp.MustCompile(`(?i)^vk_\d+$`)
)

// NormalizeUsername lowercases and trims a local username (password auth).
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// IsValidLocalUsername checks rules for password-registration usernames.
func IsValidLocalUsername(s string) bool {
	n := NormalizeUsername(s)
	if n == "" {
		return false
	}
	if vkCollision.MatchString(n) {
		return false
	}
	return usernamePattern.MatchString(n)
}
