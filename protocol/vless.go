package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type vlessModule struct {
	commonModule
}

func newVLESSModule() Module {
	return &vlessModule{commonModule: commonModule{name: "vless", supportsClients: true}}
}

func (m *vlessModule) Validate(inbound *model.Inbound) error {
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return err
	}
	if err := validateClients(settings, "clients"); err != nil {
		return err
	}
	_, err = normalizeStreamSettings(m.name, inbound.StreamSettings)
	return err
}

func (m *vlessModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	settings, err := normalizeVisionFlows(inbound.Settings)
	if err != nil {
		return nil, err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, settings, streamSettings), nil
}
