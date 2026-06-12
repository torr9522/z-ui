package protocol

import (
	"strings"
	"x-ui/database/model"
	"x-ui/xray"
)

type mixedModule struct {
	commonModule
}

func newMixedModule() Module {
	return &mixedModule{commonModule: commonModule{name: "mixed", supportsClients: false}}
}

func (m *mixedModule) Validate(inbound *model.Inbound) error {
	if strings.TrimSpace(inbound.StreamSettings) != "" && strings.TrimSpace(inbound.StreamSettings) != "{}" {
		if _, err := normalizeStreamSettings(m.name, inbound.StreamSettings); err != nil {
			return err
		}
	}
	_, err := decodeObject(inbound.Settings)
	return err
}

func (m *mixedModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	streamSettings := inbound.StreamSettings
	if strings.TrimSpace(streamSettings) != "" && strings.TrimSpace(streamSettings) != "{}" {
		normalized, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
		if err != nil {
			return nil, err
		}
		streamSettings = normalized
	}
	return buildInboundConfig(inbound, inbound.Settings, streamSettings), nil
}
