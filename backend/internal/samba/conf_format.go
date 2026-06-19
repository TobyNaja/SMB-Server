package samba

import (
	"regexp"
	"strings"
)

// invalidUserChars matches characters not allowed in samba usernames/groups.
var invalidUserChars = regexp.MustCompile(`[^\w\\@\s.\-]`)

// sanitizeUsers strips disallowed characters from each username.
func sanitizeUsers(users []string) []string {
	result := make([]string, 0, len(users))
	for _, u := range users {
		clean := strings.TrimSpace(invalidUserChars.ReplaceAllString(u, ""))
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

// parseUserList parses a smb.conf user-list value, handling quoted strings and AD backslash.
// e.g. `IT\alice @"Domain Admins" bob` → ["IT\alice", "@Domain Admins", "bob"]
func parseUserList(raw string) []string {
	if raw == "" {
		return nil
	}
	// Match @"…" or "…" or non-whitespace tokens
	re := regexp.MustCompile(`@?"[^"]+"|\S+`)
	tokens := re.FindAllString(raw, -1)
	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, strings.ReplaceAll(t, `"`, ""))
	}
	return result
}

// formatUser quotes a username for smb.conf if it contains spaces.
// Preserves the @ prefix for groups and backslash for AD users.
func formatUser(user string) string {
	prefix := ""
	name := user
	if strings.HasPrefix(user, "@") {
		prefix = "@"
		name = user[1:]
	}
	if strings.Contains(name, " ") {
		// Escape backslash inside quoted name
		safe := strings.ReplaceAll(name, `\`, `\\`)
		return prefix + `"` + safe + `"`
	}
	return prefix + name
}

// formatUserList serialises a user slice to smb.conf space-separated format.
func formatUserList(users []string) string {
	parts := make([]string, 0, len(users))
	for _, u := range users {
		if u != "" {
			parts = append(parts, formatUser(u))
		}
	}
	return strings.Join(parts, " ")
}
