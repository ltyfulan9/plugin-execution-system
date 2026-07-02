package security

import (
	"testing"
	"unicode/utf8"
)

func TestSanitizePreviewRemovesANSIAndControls(t *testing.T) {
	got := SanitizePreview("ok\x1b[31mRED\x1b[0m\x00\x07\nnext", 100)
	want := "okRED\nnext"
	if got != want {
		t.Fatalf("sanitize mismatch: got %q want %q", got, want)
	}
}

func TestSanitizePreviewTruncatesUTF8Safely(t *testing.T) {
	got := SanitizePreview("你好世界", 5)
	if !utf8.ValidString(got) {
		t.Fatalf("output should be valid utf8: %q", got)
	}
	if len(got) > 5 {
		t.Fatalf("output should be truncated")
	}
}
