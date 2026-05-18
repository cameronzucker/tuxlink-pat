package credstore_test

import (
	"testing"

	"github.com/la5nta/pat/internal/credstore"
)

func TestServiceConstant(t *testing.T) {
	if credstore.ServiceName != "tuxlink-pat" {
		t.Errorf("ServiceName = %q, want %q", credstore.ServiceName, "tuxlink-pat")
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
