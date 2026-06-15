package protocol

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"x-ui/database/model"
	"x-ui/util/json_util"
	"x-ui/xray"
)

func TestDokodemoBuildInboundGeneratesDokodemoProtocol(t *testing.T) {
	module := newDokodemoModule("dokodemo-door")
	inbound := &model.Inbound{
		Protocol: model.Protocol("Dokodemo-door"),
		Port:     23001,
		Tag:      "dokodemo-test",
		Settings: `{"address":"1.1.1.1","port":53,"network":"tcp,udp","followRedirect":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if config.Protocol != "dokodemo-door" {
		t.Fatalf("expected dokodemo-door protocol, got %q", config.Protocol)
	}
}

func TestSocksBuildInboundGeneratesSocksProtocol(t *testing.T) {
	module := newSocksModule()
	inbound := &model.Inbound{
		Protocol:       model.Socks,
		Port:           23002,
		Tag:            "socks-test",
		Settings:       `{"auth":"password","accounts":[{"user":"demo","pass":"secret"}],"udp":true,"ip":"127.0.0.1"}`,
		StreamSettings: `{"network":"ws"}`,
		Sniffing:       `{"enabled":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if config.Protocol != "socks" {
		t.Fatalf("expected socks protocol, got %q", config.Protocol)
	}
	if string(config.StreamSettings) != "{}" {
		t.Fatalf("expected socks stream settings to be cleared, got %s", string(config.StreamSettings))
	}
	if string(config.Sniffing) != "{}" {
		t.Fatalf("expected socks sniffing to be cleared, got %s", string(config.Sniffing))
	}
}

func TestHTTPBuildInboundGeneratesHTTPProtocol(t *testing.T) {
	module := newHTTPModule()
	inbound := &model.Inbound{
		Protocol:       model.Http,
		Port:           23003,
		Tag:            "http-test",
		Settings:       `{"auth":true,"accounts":[{"user":"demo","pass":"secret"}],"allowTransparent":true}`,
		StreamSettings: `{"network":"ws"}`,
		Sniffing:       `{"enabled":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if config.Protocol != "http" {
		t.Fatalf("expected http protocol, got %q", config.Protocol)
	}
	settings := decodeMap(t, string(config.Settings))
	if _, exists := settings["auth"]; exists {
		t.Fatalf("http auth must not leak into final settings: %#v", settings)
	}
	if string(config.StreamSettings) != "{}" {
		t.Fatalf("expected http stream settings to be cleared, got %s", string(config.StreamSettings))
	}
	if string(config.Sniffing) != "{}" {
		t.Fatalf("expected http sniffing to be cleared, got %s", string(config.Sniffing))
	}
}

func TestTunnelBuildInboundGeneratesTunnelProtocol(t *testing.T) {
	module := newDokodemoModule("tunnel")
	inbound := &model.Inbound{
		Protocol:       model.Tunnel,
		Port:           23004,
		Tag:            "tunnel-test",
		Settings:       `{"address":"8.8.8.8","port":53,"network":"udp"}`,
		StreamSettings: `{"network":"ws"}`,
		Sniffing:       `{"enabled":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if config.Protocol != "tunnel" {
		t.Fatalf("expected tunnel protocol, got %q", config.Protocol)
	}
	if string(config.StreamSettings) != "{}" {
		t.Fatalf("expected tunnel stream settings to be cleared, got %s", string(config.StreamSettings))
	}
	if string(config.Sniffing) != "{}" {
		t.Fatalf("expected tunnel sniffing to be cleared, got %s", string(config.Sniffing))
	}
}

func TestHTTPBuildInboundWithoutAuthOmitsAccounts(t *testing.T) {
	module := newHTTPModule()
	inbound := &model.Inbound{
		Protocol: model.Http,
		Port:     23006,
		Tag:      "http-noauth-test",
		Settings: `{"auth":false,"accounts":[{"user":"demo","pass":"secret"}],"allowTransparent":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	settings := decodeMap(t, string(config.Settings))
	if _, exists := settings["accounts"]; exists {
		t.Fatalf("http accounts must be omitted when auth is disabled: %#v", settings)
	}
	if _, exists := settings["auth"]; exists {
		t.Fatalf("http auth must not leak into final settings: %#v", settings)
	}
}

func TestSocksBuildInboundWithoutAuthOmitsAccounts(t *testing.T) {
	module := newSocksModule()
	inbound := &model.Inbound{
		Protocol: model.Socks,
		Port:     23007,
		Tag:      "socks-noauth-test",
		Settings: `{"auth":"noauth","accounts":[{"user":"demo","pass":"secret"}],"udp":true,"ip":"127.0.0.1"}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	settings := decodeMap(t, string(config.Settings))
	if _, exists := settings["accounts"]; exists {
		t.Fatalf("socks accounts must be omitted when auth is disabled: %#v", settings)
	}
}

func TestRestoredInboundConfigsPassXrayTest(t *testing.T) {
	inputs := []struct {
		module  Module
		inbound *model.Inbound
	}{
		{
			module: newDokodemoModule("dokodemo-door"),
			inbound: &model.Inbound{
				Protocol: model.Dokodemo,
				Listen:   "127.0.0.1",
				Port:     23101,
				Tag:      "dokodemo-test",
				Settings: `{"address":"1.1.1.1","port":53,"network":"tcp,udp","followRedirect":true}`,
			},
		},
		{
			module: newSocksModule(),
			inbound: &model.Inbound{
				Protocol: model.Socks,
				Listen:   "127.0.0.1",
				Port:     23102,
				Tag:      "socks-test",
				Settings: `{"auth":"password","accounts":[{"user":"demo","pass":"secret"}],"udp":true,"ip":"127.0.0.1"}`,
			},
		},
		{
			module: newHTTPModule(),
			inbound: &model.Inbound{
				Protocol: model.Http,
				Listen:   "127.0.0.1",
				Port:     23103,
				Tag:      "http-test",
				Settings: `{"accounts":[{"user":"demo","pass":"secret"}],"allowTransparent":true}`,
			},
		},
		{
			module: newMixedModule(),
			inbound: &model.Inbound{
				Protocol: model.Mixed,
				Listen:   "127.0.0.1",
				Port:     23104,
				Tag:      "mixed-test",
				Settings: `{}`,
			},
		},
		{
			module: newDokodemoModule("tunnel"),
			inbound: &model.Inbound{
				Protocol: model.Tunnel,
				Listen:   "127.0.0.1",
				Port:     23105,
				Tag:      "tunnel-test",
				Settings: `{"address":"8.8.8.8","port":53,"network":"udp"}`,
			},
		},
	}

	inbounds := make([]xray.InboundConfig, 0, len(inputs))
	for _, input := range inputs {
		cfg, err := input.module.BuildInbound(input.inbound)
		if err != nil {
			t.Fatalf("build %s inbound: %v", input.inbound.Protocol, err)
		}
		inbounds = append(inbounds, *cfg)
	}

	config := &xray.Config{
		InboundConfigs:  inbounds,
		OutboundConfigs: json_util.RawMessage(`[{"protocol":"freedom","settings":{}}]`),
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	tempFile, err := os.CreateTemp("", "xui-restored-inbounds-*.json")
	if err != nil {
		t.Fatalf("create temp config: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	xrayPath, err := filepath.Abs(filepath.Join("..", xray.GetBinaryPath()))
	if err != nil {
		t.Fatalf("resolve xray binary path: %v", err)
	}
	cmd := exec.Command(xrayPath, "-test", "-c", tempPath)
	if assetDir, err := filepath.Abs(filepath.Join("..", "bin")); err == nil {
		cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+assetDir)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("xray -test failed: %v\n%s", err, string(output))
	}
}
