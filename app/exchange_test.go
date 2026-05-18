// Copyright 2026 tuxlink-pat fork. All rights reserved.
// Use of this source code is governed by the MIT-license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/la5nta/pat/internal/credstore"
	"github.com/la5nta/wl2k-go/fbb"
	"github.com/zalando/go-keyring"
)

// Tests serialize (NO t.Parallel) because keyring.MockInit is process-global.

func TestSecureLoginCallback_PrimaryHit(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "primarypw"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	addr := fbb.AddressFromString("KK6XYZ")
	pw, err := secureLoginLookup(context.Background(), addr, &mockPromptHub{shouldNotFire: t})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pw != "primarypw" {
		t.Errorf("pw = %q, want primarypw", pw)
	}
}

func TestSecureLoginCallback_PrimaryMiss_PromptHub(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	addr := fbb.AddressFromString("KK6XYZ")
	promptHub := &mockPromptHub{returnVal: "promptedpw"}
	pw, err := secureLoginLookup(context.Background(), addr, promptHub)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pw != "promptedpw" {
		t.Errorf("pw = %q, want promptedpw", pw)
	}
	if !promptHub.fired {
		t.Errorf("promptHub did not fire")
	}
}

func TestSecureLoginCallback_SmtpProtoSkipsCredstore(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	// Pre-populate keyring with entries that would HIT if SMTP-proto wasn't skipped.
	if err := keyring.Set(credstore.ServiceName, "SOMEONE@EXAMPLE.ORG", "shouldnotreturn"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// SMTP-proto address: AddressFromString classifies non-winlink-host @-form as SMTP
	// (verified inline at fbb/message.go:564 — R3 F1 source-code-verified claim).
	addr := fbb.AddressFromString("someone@example.org")
	if addr.Proto != "SMTP" {
		t.Fatalf("setup: addr.Proto = %q, want SMTP (test assumes fbb classifies @-form-with-non-winlink-host as SMTP)", addr.Proto)
	}
	promptHub := &mockPromptHub{returnVal: "promptedpw"}
	pw, err := secureLoginLookup(context.Background(), addr, promptHub)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pw != "promptedpw" {
		t.Errorf("pw = %q, want promptedpw (SMTP-proto should skip credstore + go straight to prompt)", pw)
	}
	if !promptHub.fired {
		t.Errorf("promptHub did not fire for SMTP-proto address")
	}
}

func TestSecureLoginCallback_EmptyAddrSkipsCredstore(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	addr := fbb.Address{Addr: "", Proto: ""}
	promptHub := &mockPromptHub{returnVal: "promptedpw"}
	pw, _ := secureLoginLookup(context.Background(), addr, promptHub)
	if pw != "promptedpw" {
		t.Errorf("pw = %q, want promptedpw (empty addr should skip credstore + go to prompt)", pw)
	}
	if !promptHub.fired {
		t.Errorf("promptHub did not fire for empty addr")
	}
}

func TestSecureLoginCallback_AuxHit(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	// Future multi-account UX (or power-user manual setup) populates AuxAddr keys.
	if err := keyring.Set(credstore.ServiceName, "KK6ABC", "auxpw"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	addr := fbb.AddressFromString("KK6ABC")
	promptHub := &mockPromptHub{shouldNotFire: t}
	pw, err := secureLoginLookup(context.Background(), addr, promptHub)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pw != "auxpw" {
		t.Errorf("pw = %q, want auxpw", pw)
	}
}

func TestSecureLoginCallback_AuxMiss_PromptHub_NoFallbackToPrimary(t *testing.T) {
	// REGRESSION TEST for the dropped fallback per spec §4.7.
	// Setup: primary KK6XYZ has entry; AuxAddr KK6ABC does NOT.
	// When callback receives KK6ABC address, it MUST NOT return KK6XYZ's password.
	keyring.MockInit()
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "primarypw_should_NOT_leak"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	addr := fbb.AddressFromString("KK6ABC")
	promptHub := &mockPromptHub{returnVal: "promptedpw"}
	pw, err := secureLoginLookup(context.Background(), addr, promptHub)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pw == "primarypw_should_NOT_leak" {
		t.Errorf("REGRESSION: callback returned primary's password for AuxAddr session — the fallback should be dropped per §4.7")
	}
	if pw != "promptedpw" {
		t.Errorf("pw = %q, want promptedpw (AuxAddr miss should go to promptHub)", pw)
	}
}

func TestSecureLoginCallback_KeyringLockedFallsToPrompt(t *testing.T) {
	keyring.MockInitWithError(errors.New("default keyring is locked"))
	t.Cleanup(keyring.MockInit)
	addr := fbb.AddressFromString("KK6XYZ")
	promptHub := &mockPromptHub{returnVal: "promptedpw"}
	pw, _ := secureLoginLookup(context.Background(), addr, promptHub)
	if pw != "promptedpw" {
		t.Errorf("pw = %q, want promptedpw", pw)
	}
	if !promptHub.fired {
		t.Errorf("promptHub did not fire on locked keyring")
	}
}

// mockPromptHub is a test double for app.PromptHub. shouldNotFire marks
// tests where firing is a regression; returnVal is what fire returns.
type mockPromptHub struct {
	shouldNotFire *testing.T
	returnVal     string
	fired         bool
}

func (m *mockPromptHub) Prompt(ctx context.Context, timeout time.Duration, kind PromptKind, message string, options ...PromptOption) <-chan PromptResponse {
	if m.shouldNotFire != nil {
		m.shouldNotFire.Errorf("mockPromptHub.Prompt called; should not fire")
	}
	m.fired = true
	ch := make(chan PromptResponse, 1)
	ch <- PromptResponse{Value: m.returnVal, Err: nil}
	return ch
}
