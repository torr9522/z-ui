package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseWhitelistPorts(t *testing.T) {
	tests := []struct {
		raw  string
		want []int
	}{
		{raw: `{"ports":[22,80,443,27422]}`, want: []int{22, 80, 443, 27422}},
		{raw: `[22,80]`, want: []int{22, 80}},
		{raw: `22, 80 443`, want: []int{22, 80, 443}},
	}
	for _, test := range tests {
		got := parseWhitelistPorts(test.raw)
		if len(got) != len(test.want) {
			t.Fatalf("parseWhitelistPorts(%q) length = %d, want %d", test.raw, len(got), len(test.want))
		}
		for i := range got {
			if got[i] != test.want[i] {
				t.Fatalf("parseWhitelistPorts(%q)[%d] = %d, want %d", test.raw, i, got[i], test.want[i])
			}
		}
	}
}

func TestSSHConfigPorts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sshd_config")
	if err := os.WriteFile(path, []byte("#Port 22\nPort 2222\nPort 2200\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ports, err := sshConfigPorts(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Trim(strings.Join([]string{itoa(ports[0]), itoa(ports[1])}, ","), ",")
	if got != "2222,2200" {
		t.Fatalf("sshConfigPorts = %q", got)
	}
}

func TestParseNftBlockedPortsOnlyUsesElements(t *testing.T) {
	out := `
table inet zui_port_guard {
	set blocked_ports {
		type inet_service
		flags timeout,dynamic
		elements = { 12568 timeout 300s expires 4m23s, 443 timeout 60s }
	}
}
`
	got := parseNftBlockedPorts(out)
	if len(got) != 2 {
		t.Fatalf("blocked port count = %d, want 2: %#v", len(got), got)
	}
	if got[12568] != 263 {
		t.Fatalf("port 12568 timeout = %d, want 263", got[12568])
	}
	if got[443] != 60 {
		t.Fatalf("port 443 timeout = %d, want 60", got[443])
	}
	if _, ok := got[300]; ok {
		t.Fatalf("timeout value was parsed as a port: %#v", got)
	}
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
