package protocol

import (
	"encoding/json"
	"testing"
	"x-ui/database/model"
)

func TestVMessBuildInboundNormalizesAlterID(t *testing.T) {
	module := newVMessModule()
	inbound := &model.Inbound{
		Protocol: model.VMess,
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","alterId":64}]}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}

	settings := decodeMap(t, string(config.Settings))
	client := firstClient(t, settings)
	if client["alterId"] != float64(0) {
		t.Fatalf("expected alterId to normalize to 0, got %#v", client["alterId"])
	}
}

func TestMixedBuildInboundGeneratesMixedProtocol(t *testing.T) {
	module := newMixedModule()
	inbound := &model.Inbound{
		Protocol:       model.Mixed,
		Port:           24010,
		Tag:            "mixed-test",
		Settings:       `{}`,
		StreamSettings: `{"network":"ws"}`,
		Sniffing:       `{"enabled":true}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if config.Protocol != "mixed" {
		t.Fatalf("expected mixed protocol, got %q", config.Protocol)
	}
	if string(config.StreamSettings) != "{}" {
		t.Fatalf("mixed should not generate stream settings, got %s", string(config.StreamSettings))
	}
	if string(config.Sniffing) != "{}" {
		t.Fatalf("mixed should clear sniffing, got %s", string(config.Sniffing))
	}
}

func TestVLESSBuildInboundGeneratesRealityStream(t *testing.T) {
	module := newVLESSModule()
	inbound := &model.Inbound{
		Protocol: model.VLESS,
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111","flow":"xtls-rprx-vision"}]}`,
		StreamSettings: `{
			"network":"tcp",
			"security":"reality",
			"realitySettings":{
				"dest":"example.com:443",
				"serverNames":["example.com"],
				"privateKey":"private-key",
				"publicKey":"public-key",
				"shortIds":["abcd1234"],
				"spiderX":"/",
				"fingerprint":"chrome"
			},
			"tcpSettings":{"header":{"type":"none"}}
		}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}

	stream := decodeMap(t, string(config.StreamSettings))
	if stream["security"] != "reality" {
		t.Fatalf("expected reality security, got %#v", stream["security"])
	}
	reality, ok := stream["realitySettings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected realitySettings, got %#v", stream["realitySettings"])
	}
	if reality["dest"] != "example.com:443" ||
		reality["fingerprint"] != "chrome" ||
		reality["privateKey"] != "private-key" ||
		reality["publicKey"] != "public-key" {
		t.Fatalf("unexpected reality settings: %#v", reality)
	}
}

func TestVLESSBuildInboundGeneratesXHTTPStream(t *testing.T) {
	module := newVLESSModule()
	inbound := &model.Inbound{
		Protocol: model.VLESS,
		Settings: `{"clients":[{"id":"11111111-1111-1111-1111-111111111111"}]}`,
		StreamSettings: `{
			"network":"xhttp",
			"security":"none",
			"xhttpSettings":{
				"host":"example.com",
				"path":"/xhttp",
				"mode":"stream-one",
				"headers":{"X-Test":"1"}
			}
		}`,
	}

	config, err := module.BuildInbound(inbound)
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}

	stream := decodeMap(t, string(config.StreamSettings))
	if stream["network"] != "xhttp" {
		t.Fatalf("expected xhttp network, got %#v", stream["network"])
	}
	xhttp, ok := stream["xhttpSettings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected xhttpSettings, got %#v", stream["xhttpSettings"])
	}
	if xhttp["host"] != "example.com" || xhttp["path"] != "/xhttp" || xhttp["mode"] != "stream-one" {
		t.Fatalf("unexpected xhttp settings: %#v", xhttp)
	}
	if _, ok := xhttp["headers"].(map[string]interface{}); !ok {
		t.Fatalf("expected xhttp extra headers to be preserved, got %#v", xhttp)
	}
}

func TestVMessValidateRejectsInvalidUUID(t *testing.T) {
	module := newVMessModule()
	inbound := &model.Inbound{
		Protocol: model.VMess,
		Settings: `{"clients":[{"id":"not-a-uuid"}]}`,
	}
	if err := module.Validate(inbound); err == nil {
		t.Fatal("expected invalid vmess uuid to fail validation")
	}
}

func TestVLESSValidateRejectsMissingRealityFields(t *testing.T) {
	module := newVLESSModule()
	inbound := &model.Inbound{
		Protocol:       model.VLESS,
		Settings:       `{"clients":[{"id":"11111111-1111-1111-1111-111111111111"}]}`,
		StreamSettings: `{"network":"tcp","security":"reality","realitySettings":{"privateKey":"","dest":""}}`,
	}
	if err := module.Validate(inbound); err == nil {
		t.Fatal("expected missing reality fields to fail validation")
	}
}

func TestTrojanValidateRejectsEmptyPassword(t *testing.T) {
	module := newTrojanModule()
	inbound := &model.Inbound{
		Protocol: model.Trojan,
		Settings: `{"clients":[{"password":""}]}`,
	}
	if err := module.Validate(inbound); err == nil {
		t.Fatal("expected empty trojan password to fail validation")
	}
}

func decodeMap(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode json: %v (%s)", err, raw)
	}
	return out
}

func firstClient(t *testing.T, settings map[string]interface{}) map[string]interface{} {
	t.Helper()
	clients, ok := settings["clients"].([]interface{})
	if !ok || len(clients) == 0 {
		t.Fatalf("expected clients, got %#v", settings["clients"])
	}
	client, ok := clients[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected client object, got %#v", clients[0])
	}
	return client
}
