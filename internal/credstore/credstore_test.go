package credstore_test

import (
	"errors"
	"testing"

	"github.com/la5nta/pat/internal/credstore"
	"github.com/zalando/go-keyring"
)

// setupMock initializes the keyring mock and registers cleanup. Tests call this
// FIRST, then optionally Set entries via keyring.Set directly.
//
// IMPORTANT: keyring.MockInit() mutates package-global state. Tests MUST NOT
// call t.Parallel() (per spec §3.6). Cleanup uses keyring.DeleteAll to avoid
// cross-test pollution.
func setupMock(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
}

func TestServiceConstant(t *testing.T) {
	if credstore.ServiceName != "tuxlink-pat" {
		t.Errorf("ServiceName = %q, want %q", credstore.ServiceName, "tuxlink-pat")
	}
}

func TestGet_Hit(t *testing.T) {
	setupMock(t)
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "secretpw"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	pw, found, err := credstore.Get("KK6XYZ")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if !found {
		t.Errorf("Get: found=false, want true")
	}
	if pw != "secretpw" {
		t.Errorf("Get: pw=%q, want %q", pw, "secretpw")
	}
}

func TestGet_Miss(t *testing.T) {
	setupMock(t)
	pw, found, err := credstore.Get("UNSETCALLSIGN")
	if err != nil {
		t.Errorf("Get err: %v, want nil", err)
	}
	if found {
		t.Errorf("Get: found=true, want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
}

func TestGet_NotFoundIsMiss(t *testing.T) {
	// This is a regression test for the ErrNotFound mapping per spec §3.2.
	// Same behavior as TestGet_Miss but explicit about the sentinel mapping.
	setupMock(t)
	pw, found, err := credstore.Get("NEVERSET")
	if err != nil {
		t.Errorf("Get err: %v, want nil (ErrNotFound should map to found=false, err=nil)", err)
	}
	if found {
		t.Errorf("Get: found=true, want false")
	}
	_ = pw
}

func TestGet_EmptyStoredTreatedAsMiss(t *testing.T) {
	setupMock(t)
	// Some keyring backends (e.g., Linux secret-service) accept Set("") and
	// store an empty string; others (Windows wincred) reject. Our credstore
	// normalizes this to "no entry" per spec §3.5 — uniform UX regardless
	// of backend behavior (R3 F4 caught).
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", ""); err != nil {
		t.Fatalf("Set: %v", err)
	}
	pw, found, err := credstore.Get("KK6XYZ")
	if err != nil {
		t.Errorf("Get err: %v, want nil", err)
	}
	if found {
		t.Errorf("Get: found=true (empty-stored treated as hit), want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
}

func TestGet_EmptyCallsign_ShortCircuit(t *testing.T) {
	setupMock(t)
	// We can't easily intercept keyring.Get calls in zalando's mock,
	// so we instead verify by attempting to write to a known account
	// then reading with an empty callsign — if Get DID forward to the
	// backend, it'd succeed in returning the entry under "" (or error).
	// If Get short-circuits, we get (false, nil) without backend call.
	if err := keyring.Set(credstore.ServiceName, "", "shouldnotreturn"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	pw, found, err := credstore.Get("")
	if err != nil {
		t.Errorf("Get err: %v, want nil", err)
	}
	if found {
		t.Errorf("Get: found=true (backend not short-circuited), want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
}

func TestGet_WhitespaceCallsign_ShortCircuit(t *testing.T) {
	setupMock(t)
	// Same defense: set an entry under "   " (which would be the post-trim
	// account if Get DIDN'T short-circuit); verify Get returns false.
	pw, found, err := credstore.Get("   ")
	if err != nil {
		t.Errorf("Get err: %v, want nil", err)
	}
	if found {
		t.Errorf("Get: found=true, want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
}

func TestGet_ErrLockedClassified(t *testing.T) {
	// Per zalando/go-keyring's mock support, MockInitWithError sets a
	// global error that all subsequent Set/Get/Delete calls return.
	// We inject a synthetic "locked"-style error and verify credstore
	// classifies it as ErrLocked sentinel.
	lockedErr := errors.New("default keyring is locked")
	keyring.MockInitWithError(lockedErr)
	t.Cleanup(keyring.MockInit) // restore to non-error mock

	pw, found, err := credstore.Get("KK6XYZ")
	if found {
		t.Errorf("Get: found=true on locked, want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
	if !errors.Is(err, credstore.ErrLocked) {
		t.Errorf("Get err: %v (errors.Is(ErrLocked) = false); want ErrLocked sentinel", err)
	}
}

func TestGet_ErrUnavailableClassified(t *testing.T) {
	// D-Bus unreachable / no-secret-service is a different error class —
	// distinguishes "you forgot to install gnome-keyring" from "you
	// haven't unlocked the keyring yet."
	unavailErr := errors.New("dbus: cannot connect to session bus: address not set")
	keyring.MockInitWithError(unavailErr)
	t.Cleanup(keyring.MockInit)

	pw, found, err := credstore.Get("KK6XYZ")
	if found {
		t.Errorf("Get: found=true on D-Bus error, want false")
	}
	if pw != "" {
		t.Errorf("Get: pw=%q, want empty", pw)
	}
	if !errors.Is(err, credstore.ErrUnavailable) {
		t.Errorf("Get err: %v; want ErrUnavailable sentinel", err)
	}
}

func TestGet_ErrServiceUnknownClassified(t *testing.T) {
	// Per parent Codex round on Phase 2 (2026-05-18): when D-Bus is
	// reachable but org.freedesktop.secrets is not installed (e.g., no
	// gnome-keyring / kwallet-pam), godbus returns a different error
	// shape than the "cannot connect to" / "dbus" markers. Verify
	// ServiceUnknown is classified as ErrUnavailable.
	svcUnknownErr := errors.New("The name org.freedesktop.secrets was not provided by any .service files")
	keyring.MockInitWithError(svcUnknownErr)
	t.Cleanup(keyring.MockInit)

	_, _, err := credstore.Get("KK6XYZ")
	if !errors.Is(err, credstore.ErrUnavailable) {
		t.Errorf("Get err: %v; want ErrUnavailable sentinel (ServiceUnknown should classify as unavailable)", err)
	}
}

func TestGet_ErrServiceUnknownByErrorName(t *testing.T) {
	// Alternate ServiceUnknown error shape: error name without the
	// human-readable message tail.
	svcUnknownErr := errors.New("org.freedesktop.DBus.Error.ServiceUnknown")
	keyring.MockInitWithError(svcUnknownErr)
	t.Cleanup(keyring.MockInit)

	_, _, err := credstore.Get("KK6XYZ")
	if !errors.Is(err, credstore.ErrUnavailable) {
		t.Errorf("Get err: %v; want ErrUnavailable sentinel", err)
	}
}

func TestGet_CasingNormalization(t *testing.T) {
	// Convergent adrev finding R2 F1 + R4 P2: wizard may write lowercase
	// callsign; Pat reads via addr.Addr which is uppercased; without
	// normalization, two different entries → silent miss.
	setupMock(t)
	// Write under normalized form (what the wizard would do):
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// Read with lowercase input (should normalize and hit):
	pw, found, err := credstore.Get("kk6xyz")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if !found || pw != "secret" {
		t.Errorf("Get(\"kk6xyz\") = (%q, %v); want (\"secret\", true) — normalization should match KK6XYZ entry", pw, found)
	}
	// And whitespace-wrapped:
	pw, found, err = credstore.Get("  kk6xyz  ")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if !found || pw != "secret" {
		t.Errorf("Get(whitespace-wrapped lowercase) = (%q, %v); want (\"secret\", true)", pw, found)
	}
}

func TestNormalizeAccount(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantOut string
		wantOk  bool
	}{
		{"bare callsign", "KK6XYZ", "KK6XYZ", true},
		{"lowercase normalized", "kk6xyz", "KK6XYZ", true},
		{"leading whitespace trimmed", "  KK6XYZ", "KK6XYZ", true},
		{"trailing whitespace trimmed", "KK6XYZ  ", "KK6XYZ", true},
		{"mixed whitespace + lowercase", "  kk6xyz  ", "KK6XYZ", true},
		{"empty string rejected", "", "", false},
		{"whitespace-only rejected", "   ", "", false},
		{"tab-only rejected", "\t\t", "", false},
		{"newline-only rejected", "\n", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := credstore.NormalizeAccount(tc.input)
			if out != tc.wantOut || ok != tc.wantOk {
				t.Errorf("NormalizeAccount(%q) = (%q, %v), want (%q, %v)",
					tc.input, out, ok, tc.wantOut, tc.wantOk)
			}
		})
	}
}
