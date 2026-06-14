package protocol

import (
	"strings"
	"testing"
	"x-ui/database/model"
)

func TestVLESSBuildInboundAddsDefaultDecryption(t *testing.T) {
	module := newVLESSModule()
	inbound := &model.Inbound{
		Protocol: model.VLESS,
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111"}]}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if !strings.Contains(string(config.Settings), `"decryption":"none"`) {
		t.Fatalf("expected decryption:none in settings: %s", string(config.Settings))
	}
}

func TestVLESSLegacyOriginFlowMigratesToVision(t *testing.T) {
	assertVLESSFlow(t, "xtls-rprx-origin", `"flow":"xtls-rprx-vision"`)
}

func TestVLESSLegacyDirectFlowMigratesToVision(t *testing.T) {
	assertVLESSFlow(t, "xtls-rprx-direct", `"flow":"xtls-rprx-vision"`)
}

func TestVLESSLegacySpliceFlowIsRemoved(t *testing.T) {
	assertVLESSFlow(t, "xtls-rprx-splice", `"flow"`)
}

func assertVLESSFlow(t *testing.T, inputFlow string, expected string) {
	t.Helper()
	module := newVLESSModule()
	inbound := &model.Inbound{
		Protocol: model.VLESS,
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","flow":"` + inputFlow + `"}]}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	settings := string(config.Settings)
	if inputFlow == "xtls-rprx-splice" {
		if strings.Contains(settings, expected) {
			t.Fatalf("expected splice flow to be removed, got %s", settings)
		}
		return
	}
	if !strings.Contains(settings, expected) {
		t.Fatalf("expected %s after normalizing %s, got %s", expected, inputFlow, settings)
	}
}
