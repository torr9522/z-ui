package protocol

import (
	"strings"
	"x-ui/database/model"
	"x-ui/xray"
)

type passthroughModule struct {
	commonModule
}

func newPassthroughModule(name string, supportsClients bool) Module {
	return &passthroughModule{commonModule: commonModule{name: name, supportsClients: supportsClients}}
}

func (m *passthroughModule) Validate(inbound *model.Inbound) error {
	_, err := decodeObject(inbound.Settings)
	if err != nil {
		return err
	}
	if strings.TrimSpace(inbound.StreamSettings) != "" && strings.TrimSpace(inbound.StreamSettings) != "{}" {
		_, err = normalizeStreamSettings(m.name, inbound.StreamSettings)
	}
	return err
}

func (m *passthroughModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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
