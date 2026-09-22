package sanitize

import (
	"html"
	"strings"
)

// Text trims spaces, strips null bytes, and escapes HTML characters to prevent XSS
func Text(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.TrimSpace(s)
	return html.EscapeString(s)
}

// Email normalizes email addresses by trimming and lowercasing
func Email(s string) string {
	s = strings.TrimSpace(s)
	return strings.ToLower(s)
}
