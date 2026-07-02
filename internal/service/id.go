package service

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "") + "_" + hex.EncodeToString(b)
}

// NewPublicID exposes the same sortable ID style for infrastructure adapters
// that must create attempt/event/audit records without importing service internals.
func NewPublicID(prefix string) string { return newID(prefix) }
