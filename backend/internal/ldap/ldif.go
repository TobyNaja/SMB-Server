package ldap

import (
	"encoding/base64"
	"strings"
)

// Entry is a single LDAP record (attribute name → string or []string).
type Entry map[string]interface{}

// ParseLDIF parses ldapsearch -o ldif-wrap=no output.
// Handles base64 values (key:: value), multi-line continuation (leading space),
// and repeated attributes (collapsed to []string).
func ParseLDIF(output string) []Entry {
	var entries []Entry
	current := Entry{}
	lastKey := ""

	for _, raw := range strings.Split(output, "\n") {
		// Multi-line continuation: value wrapped onto next line with a leading space.
		// Must concatenate (not append as a new value).
		if strings.HasPrefix(raw, " ") && lastKey != "" {
			suffix := strings.TrimPrefix(raw, " ")
			switch existing := current[lastKey].(type) {
			case string:
				current[lastKey] = existing + suffix
			case []string:
				if len(existing) > 0 {
					existing[len(existing)-1] += suffix
					current[lastKey] = existing
				}
			}
			continue
		}

		line := strings.TrimSpace(raw)

		// Blank line = end of entry
		if line == "" {
			if len(current) > 0 {
				if _, hasDN := current["dn"]; hasDN {
					entries = append(entries, current)
				}
			}
			current = Entry{}
			lastKey = ""
			continue
		}

		// Comment
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Base64 value: key:: base64data
		if idx := strings.Index(line, ":: "); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(line[:idx]))
			encoded := strings.TrimSpace(line[idx+3:])
			decoded := decodeBase64(encoded)
			lastKey = key
			appendOrConcat(current, key, decoded, false)
			continue
		}

		// Plain value: key: value
		if idx := strings.Index(line, ": "); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(line[:idx]))
			val := strings.TrimSpace(line[idx+2:])
			lastKey = key
			appendOrConcat(current, key, val, false)
		}
	}

	// Flush last entry
	if len(current) > 0 {
		if _, hasDN := current["dn"]; hasDN {
			entries = append(entries, current)
		}
	}

	return entries
}

// GetString extracts a scalar string value from an entry, preferring the first element of slices.
func GetString(e Entry, key string) string {
	v, ok := e[key]
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}
	return ""
}

// GetStrings extracts a multi-value attribute as a []string.
func GetStrings(e Entry, key string) []string {
	v, ok := e[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case string:
		return []string{t}
	case []string:
		return t
	}
	return nil
}

func appendOrConcat(e Entry, key, val string, replace bool) {
	existing, exists := e[key]
	if !exists || replace {
		e[key] = val
		return
	}
	switch t := existing.(type) {
	case string:
		// Second occurrence → promote to slice
		e[key] = []string{t, val}
	case []string:
		e[key] = append(t, val)
	}
}

func decodeBase64(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return strings.TrimRight(string(b), "\x00")
}
