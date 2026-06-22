package protocol

import (
	"encoding/json"
	"testing"
	"x-ui/database/model"
)

func TestShadowsocksRebuildDropsResidualTLSState(t *testing.T) {
	inbound := &model.Inbound{
		Protocol: model.Shadowsocks,
		Port:     25001,
		Tag:      "ss-rebuild-test",
		Settings: `{"clients":[{"password":"legacy"}],"method":"chacha20-poly1305","network":"tcp,udp","password":"fresh-pass"}`,
		StreamSettings: `{
			"network":"tcp",
			"security":"tls",
			"tlsSettings":{
				"serverName":"example.com",
				"certificates":[{"certificateFile":"/tmp/test.crt","keyFile":"/tmp/test.key"}]
			}
		}`,
	}

	schema, err := BuildSchemaFromInbound(inbound, "shadowsocks", true)
	if err != nil {
		t.Fatalf("build schema: %v", err)
	}
	cfg, err := BuildInboundByProtocol(schema)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}

	stream := decodeFactoryMap(t, string(cfg.StreamSettings))
	if len(stream) != 0 {
		t.Fatalf("expected residual stream settings to be removed, got %#v", stream)
	}
	settings := decodeFactoryMap(t, string(cfg.Settings))
	if settings["password"] != "fresh-pass" {
		t.Fatalf("expected rebuilt password, got %#v", settings["password"])
	}
	if _, exists := settings["clients"]; exists {
		t.Fatalf("expected shadowsocks rebuild to remove clients payload: %#v", settings)
	}
}

func TestDokodemoRebuildProducesExplicitSettings(t *testing.T) {
	inbound := &model.Inbound{
		Protocol:       model.Dokodemo,
		Listen:         "127.0.0.1",
		Port:           25002,
		Tag:            "dokodemo-rebuild-test",
		Settings:       `{"address":"93.184.216.34","port":80,"network":"tcp","followRedirect":false}`,
		StreamSettings: `{"network":"ws","security":"tls"}`,
	}

	schema, err := BuildSchemaFromInbound(inbound, "dokodemo-door", true)
	if err != nil {
		t.Fatalf("build schema: %v", err)
	}
	cfg, err := BuildDokodemoInbound(schema)
	if err != nil {
		t.Fatalf("build dokodemo inbound: %v", err)
	}

	if cfg.Protocol != "dokodemo-door" {
		t.Fatalf("expected dokodemo-door protocol, got %q", cfg.Protocol)
	}
	settings := decodeFactoryMap(t, string(cfg.Settings))
	if settings["address"] != "93.184.216.34" {
		t.Fatalf("expected explicit address, got %#v", settings["address"])
	}
	if settings["network"] != "tcp" {
		t.Fatalf("expected explicit tcp network, got %#v", settings["network"])
	}
	if int(settings["port"].(float64)) != 80 {
		t.Fatalf("expected explicit port 80, got %#v", settings["port"])
	}
	stream := decodeFactoryMap(t, string(cfg.StreamSettings))
	if len(stream) != 0 {
		t.Fatalf("expected dokodemo rebuild to clear stream settings, got %#v", stream)
	}
}

func decodeFactoryMap(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode json: %v (%s)", err, raw)
	}
	return out
}
