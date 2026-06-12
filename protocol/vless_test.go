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
