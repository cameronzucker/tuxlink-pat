// Package credstore wraps github.com/zalando/go-keyring with tuxlink-pat-specific
// conventions for the WL2K password lookup path. See
// docs/superpowers/specs/2026-05-18-cred-handling-design.md in the tuxlink repo
// for full design rationale.
package credstore

import "strings"

// ServiceName is the OS-keyring service-string under which tuxlink-pat stores
// its WL2K credentials. Convention per spec §4.2: hardcoded fork-namespaced
// service + per-callsign account (matches GitHub CLI / aws-vault / HashiCorp
// Vault prior-art).
const ServiceName = "tuxlink-pat"

// NormalizeAccount returns the canonical keyring account string for a callsign:
// strings.ToUpper(strings.TrimSpace(callsign)). Returns ("", false) for empty
// or whitespace-only inputs — callers MUST treat (false) as "no lookup;
// fall through to prompt or error per call-site rules" per spec §3.3.
//
// Both the writer (tuxlink wizard, per tuxlink-ko0) and the reader (this
// package) MUST apply this normalization to avoid silent miss caused by
// case differences (R2 F1 + R3 F1 + R4 P2 — convergent adrev finding).
func NormalizeAccount(callsign string) (string, bool) {
	trimmed := strings.TrimSpace(callsign)
	if trimmed == "" {
		return "", false
	}
	return strings.ToUpper(trimmed), true
}
