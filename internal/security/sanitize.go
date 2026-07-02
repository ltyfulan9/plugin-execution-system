package security

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

// SanitizePreview removes terminal control sequences and truncates text for safe UI/log display.
// It intentionally preserves common whitespace so multi-line stderr remains readable.
func SanitizePreview(input string, maxBytes int) string {
	if input == "" {
		return ""
	}
	clean := ansiEscapeRE.ReplaceAllString(input, "")
	var b strings.Builder
	for _, r := range clean {
		if r == '\n' || r == '\r' || r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
		if maxBytes > 0 && b.Len() >= maxBytes {
			break
		}
	}
	out := b.String()
	if maxBytes > 0 && len(out) > maxBytes {
		out = out[:maxBytes]
		for !utf8.ValidString(out) && len(out) > 0 {
			out = out[:len(out)-1]
		}
	}
	return out
}
