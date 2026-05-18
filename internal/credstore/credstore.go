// Package credstore wraps github.com/zalando/go-keyring with tuxlink-pat-specific
// conventions for the WL2K password lookup path. See
// docs/superpowers/specs/2026-05-18-cred-handling-design.md in the tuxlink repo
// for full design rationale.
package credstore

import (
	"errors"
	"strings"

	"github.com/zalando/go-keyring"
)

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

// Get looks up the WL2K password for the given callsign in the OS keyring.
//
// Returns:
//   - (pw, true, nil) on hit with a non-empty stored password.
//   - ("", false, nil) on:
//   - empty/whitespace-only callsign (no backend call; short-circuit)
//   - keyring entry not found (keyring.ErrNotFound mapped to clean miss)
//   - empty-string stored password (treated as miss per spec §3.2 / R3 F4)
//   - ("", false, ErrLocked) when the keyring is locked (operator needs to unlock).
//   - ("", false, ErrUnavailable) when D-Bus is unreachable or secret-service
//     is not installed.
//   - ("", false, <other err>) for unclassified errors (caller logs + falls
//     through per per-call-site rules in spec §3.5).
func Get(callsign string) (string, bool, error) {
	account, ok := NormalizeAccount(callsign)
	if !ok {
		return "", false, nil
	}
	pw, err := keyring.Get(ServiceName, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", false, nil
		}
		return "", false, classifyErr(err)
	}
	if pw == "" {
		return "", false, nil
	}
	return pw, true, nil
}

// classifyErr stub for now; full classification added in Task 2.11.
func classifyErr(err error) error { return err }
