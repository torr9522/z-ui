package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type vmessModule struct {
	commonModule
}

func newVMessModule() Module {
	return &vmessModule{commonModule: commonModule{name: "vmess", supportsClients: true}}
}

func (m *vmessModule) Validate(inbound *model.Inbound) error {
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

func (m *vmessModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, streamSettings), nil
}
