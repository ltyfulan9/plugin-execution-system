package validator

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func ValidateJSONObject(v map[string]any) bool { return v != nil }

func NormalizeJSON(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func HashJSON(v any) (string, error) {
	b, err := NormalizeJSON(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func HashStrings(items []string) string {
	cp := append([]string(nil), items...)
	sort.Strings(cp)
	b, _ := json.Marshal(cp)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func LimitJSONSize(raw []byte, max int) bool { return len(raw) <= max }
