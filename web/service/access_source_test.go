package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseAccessSourceLineIPv4(t *testing.T) {
	line := "2026/06/21 19:00:01.123456 from tcp:203.0.113.10:54321 accepted tcp:example.com:443 [inbound-12568] email: user"

	event, ok := parseAccessSourceLine(line)
	if !ok {
		t.Fatal("expected line to parse")
	}
	if event.IP != "203.0.113.10" {
		t.Fatalf("ip = %q, want 203.0.113.10", event.IP)
	}
	if event.Port != 12568 {
		t.Fatalf("port = %d, want 12568", event.Port)
	}
	if event.Timestamp == 0 {
		t.Fatal("expected timestamp")
	}
}

func TestParseAccessSourceLineIPv6(t *testing.T) {
	line := "2026/06/21 19:00:01.123456 from tcp:[2001:db8::1]:54321 accepted udp:example.com:443 [inbound-12568] email: user"

	event, ok := parseAccessSourceLine(line)
	if !ok {
		t.Fatal("expected line to parse")
	}
	if event.IP != "2001:db8::1" {
		t.Fatalf("ip = %q, want 2001:db8::1", event.IP)
	}
	if event.Port != 12568 {
		t.Fatalf("port = %d, want 12568", event.Port)
	}
}

func TestParseAccessSourceLineFiltersLoopback(t *testing.T) {
	line := "2026/06/21 19:00:01.123456 from tcp:127.0.0.1:54321 accepted tcp:example.com:443 [inbound-12568] email: user"

	if _, ok := parseAccessSourceLine(line); ok {
		t.Fatal("expected loopback access source to be ignored")
	}
}

func TestBuildPortUsageStats(t *testing.T) {
	events := []*accessSourceEvent{
		{IP: "203.0.113.10", Port: 12568, Timestamp: 100},
		{IP: "203.0.113.10", Port: 12568, Timestamp: 120},
		{IP: "203.0.113.11", Port: 12568, Timestamp: 110},
		{IP: "203.0.113.12", Port: 12569, Timestamp: 90},
	}

	ports := buildPortUsageStats(events)
	if len(ports) != 2 {
		t.Fatalf("ports length = %d, want 2", len(ports))
	}

	port12568 := findPortUsageStat(ports, 12568)
	if port12568 == nil {
		t.Fatal("missing port 12568 stats")
	}
	if port12568.UniqueIPs != 2 {
		t.Fatalf("unique ips = %d, want 2", port12568.UniqueIPs)
	}
	if port12568.TotalHits != 3 {
		t.Fatalf("total hits = %d, want 3", port12568.TotalHits)
	}
	if port12568.LastSeen != 120 {
		t.Fatalf("last seen = %d, want 120", port12568.LastSeen)
	}

	port12569 := findPortUsageStat(ports, 12569)
	if port12569 == nil {
		t.Fatal("missing port 12569 stats")
	}
	if port12569.UniqueIPs != 1 {
		t.Fatalf("unique ips = %d, want 1", port12569.UniqueIPs)
	}
	if port12569.TotalHits != 1 {
		t.Fatalf("total hits = %d, want 1", port12569.TotalHits)
	}
	if len(port12568.IPs) != 2 {
		t.Fatalf("port ip stats length = %d, want 2", len(port12568.IPs))
	}
	if port12568.IPs[0].IP == "" || port12568.IPs[0].Geo == "" || port12568.IPs[0].ISP == "" {
		t.Fatalf("expected ip profile fields, got %+v", port12568.IPs[0])
	}
}

func TestAccessSourceStateReadsOnlyNewLogLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.log")
	initial := "" +
		"2026/06/21 19:00:01.123456 from tcp:203.0.113.10:54321 accepted tcp:example.com:443 [inbound-12568] email: user\n" +
		"2026/06/21 19:00:02.123456 from tcp:203.0.113.11:54322 accepted tcp:example.com:443 [inbound-12568] email: user\n"
	if err := os.WriteFile(logPath, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	state := newAccessSourceState()
	state.resolver = staticIPProfileResolver{}
	report, exists, err := state.analyze(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected log file to exist")
	}
	port := findPortUsageStat(report.Ports, 12568)
	if port == nil || port.UniqueIPs != 2 || port.TotalHits != 2 {
		t.Fatalf("initial stats = %+v, want 2 unique ips and 2 hits", port)
	}

	appended := "2026/06/21 19:00:03.123456 from tcp:203.0.113.12:54323 accepted tcp:example.com:443 [inbound-12568] email: user\n"
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(appended); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	cachedReport, _, err := state.analyze(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cachedPort := findPortUsageStat(cachedReport.Ports, 12568)
	if cachedPort == nil || cachedPort.UniqueIPs != 2 || cachedPort.TotalHits != 2 {
		t.Fatalf("batch cached stats = %+v, want unchanged stats before batch interval", cachedPort)
	}

	state.lastRun = time.Now().Add(-accessSourceBatchEvery - time.Second)
	report, _, err = state.analyze(logPath)
	if err != nil {
		t.Fatal(err)
	}
	port = findPortUsageStat(report.Ports, 12568)
	if port == nil || port.UniqueIPs != 3 || port.TotalHits != 3 {
		t.Fatalf("incremental stats = %+v, want 3 unique ips and 3 hits", port)
	}
}

func TestIPProfileResolverCachesUntilTTL(t *testing.T) {
	now := time.Unix(1000, 0)
	calls := 0
	resolver := newCachingIPProfileResolver(time.Hour, func(ip string) IPProfile {
		calls++
		return IPProfile{IP: ip, Geo: "CN/广东/深圳", ISP: "China Mobile", DeviceType: "MOBILE"}
	})
	resolver.now = func() time.Time { return now }

	first := resolver.Resolve("203.0.113.10")
	second := resolver.Resolve("203.0.113.10")
	if calls != 1 {
		t.Fatalf("lookup calls = %d, want 1", calls)
	}
	if first != second {
		t.Fatalf("cached profile mismatch: first=%+v second=%+v", first, second)
	}

	now = now.Add(2 * time.Hour)
	_ = resolver.Resolve("203.0.113.10")
	if calls != 2 {
		t.Fatalf("lookup calls after ttl = %d, want 2", calls)
	}
}

func TestCleanISPNameAndDeviceType(t *testing.T) {
	cases := []struct {
		organization string
		isp          string
		deviceType   string
	}{
		{"China Mobile Communications Group Co., Ltd.", "China Mobile", "MOBILE"},
		{"China Telecom", "China Telecom", "HOME_BROADBAND"},
		{"Amazon.com, Inc.", "AWS", "CLOUD_SERVER"},
		{"Oracle Cloud", "Oracle", "CLOUD_SERVER"},
		{"Example VPS Hosting Server", "Example VPS Hosting", "DATACENTER"},
		{"", "未知", "UNKNOWN"},
	}
	for _, tc := range cases {
		if got := cleanISPName(tc.organization); got != tc.isp {
			t.Fatalf("cleanISPName(%q) = %q, want %q", tc.organization, got, tc.isp)
		}
		if got := inferDeviceType(tc.organization); got != tc.deviceType {
			t.Fatalf("inferDeviceType(%q) = %q, want %q", tc.organization, got, tc.deviceType)
		}
	}
}

func findPortUsageStat(ports []*PortUsageStat, port int) *PortUsageStat {
	for _, stats := range ports {
		if stats.Port == port {
			return stats
		}
	}
	return nil
}
