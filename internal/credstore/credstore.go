// Package credstore wraps github.com/zalando/go-keyring with tuxlink-pat-specific
// conventions for the WL2K password lookup path. See
// docs/superpowers/specs/2026-05-18-cred-handling-design.md in the tuxlink repo
// for full design rationale.
package credstore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

// Exported error sentinels for caller error-classification per spec §3.5.
// Callers use errors.Is(err, ErrLocked) / errors.Is(err, ErrUnavailable)
// to dispatch on log level and fallback behavior.
var (
	// ErrLocked indicates the OS keyring is locked (operator must unlock via
	// OS tools — Seahorse, Keychain Access, etc.). Interactive callers log
	// at WARN level and fall through to promptHub.
	ErrLocked = errors.New("credstore: keyring locked")

	// ErrUnavailable indicates D-Bus is unreachable or secret-service is not
	// installed. Interactive callers log at ERROR level and fall through to
	// promptHub. Configuration problem; operator needs to install
	// gnome-keyring / kwallet-pam or equivalent.
	ErrUnavailable = errors.New("credstore: keyring backend unavailable")
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

// classifyErr maps zalando/go-keyring's per-backend errors to our exported
// sentinels by matching substrings in the error string. Per spec §3.2 + R3 F7
// (cross-platform ErrNotFound mapping is not contractually guaranteed; we
// classify defensively).
func classifyErr(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	// Linux secret-service / gnome-keyring "locked" markers:
	if strings.Contains(s, "is locked") || strings.Contains(s, "Locked") {
		return fmt.Errorf("%w: %v", ErrLocked, err)
	}
	// D-Bus connection markers (Linux):
	if strings.Contains(s, "cannot connect to") || strings.Contains(s, "dbus") {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	// Unclassified; return raw for caller logging.
	return err
}
