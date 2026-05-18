//go:build integration

package credstore_test

import (
	"testing"

	"github.com/la5nta/pat/internal/credstore"
	"github.com/zalando/go-keyring"
)

func TestRealKeyring_RoundTrip(t *testing.T) {
	t.Cleanup(func() { _ = keyring.DeleteAll(credstore.ServiceName) })

	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "integration-test-pw"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	pw, found, err := credstore.Get("KK6XYZ")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if !found {
		t.Errorf("Get: found=false, want true")
	}
	if pw != "integration-test-pw" {
		t.Errorf("Get: pw=%q, want %q", pw, "integration-test-pw")
	}
}

func TestRealKeyring_DeleteCleanup(t *testing.T) {
	if err := keyring.Set(credstore.ServiceName, "KK6XYZ", "to-be-deleted"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := keyring.DeleteAll(credstore.ServiceName); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	_, found, err := credstore.Get("KK6XYZ")
	if err != nil {
		t.Errorf("Get post-delete err: %v, want nil", err)
	}
	if found {
		t.Errorf("Get post-delete: found=true, want false")
	}
}
