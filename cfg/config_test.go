// Copyright 2026 tuxlink-mib authors. All rights reserved.
// Use of this source code is governed by the MIT-license that can be
// found in the LICENSE file.

package cfg_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/la5nta/pat/cfg"
)

func TestConfigParse_LegacyAuxAddrPasswordStripped(t *testing.T) {
	// Legacy form per Pat <1.0.0: auxiliary_addresses entries may contain
	// "CALL:password" (the Address:Password marshal form). Post-cred-refactor
	// (per spec §3.2), we drop AuxAddr.Password but PRESERVE the custom
	// UnmarshalJSON to strip the colon-suffix and never re-emit the password.
	const legacyConfigJSON = `{
		"mycall": "KK6XYZ",
		"auxiliary_addresses": ["KK6ABC:legacypw", "KK6DEF"]
	}`
	var c cfg.Config
	if err := json.Unmarshal([]byte(legacyConfigJSON), &c); err != nil {
		t.Fatalf("Unmarshal failed on legacy config: %v", err)
	}
	if len(c.AuxAddrs) != 2 {
		t.Fatalf("AuxAddrs len = %d, want 2", len(c.AuxAddrs))
	}
	// Password portion stripped:
	if c.AuxAddrs[0].Address != "KK6ABC" {
		t.Errorf("AuxAddrs[0].Address = %q, want %q (password should be stripped)", c.AuxAddrs[0].Address, "KK6ABC")
	}
	if c.AuxAddrs[1].Address != "KK6DEF" {
		t.Errorf("AuxAddrs[1].Address = %q, want %q", c.AuxAddrs[1].Address, "KK6DEF")
	}
	// Re-marshal: verify NO password ever re-emitted:
	out, err := json.Marshal(c.AuxAddrs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(out), "legacypw") {
		t.Errorf("Re-marshal leaked password: %s", out)
	}
	if strings.Contains(string(out), ":") {
		t.Errorf("Re-marshal includes colon-form: %s; want plain string form [\"KK6ABC\",\"KK6DEF\"]", out)
	}
}
