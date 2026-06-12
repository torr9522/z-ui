package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type trojanModule struct {
	commonModule
}

func newTrojanModule() Module {
	return &trojanModule{commonModule: commonModule{name: "trojan", supportsClients: true}}
}

func (m *trojanModule) Validate(inbound *model.Inbound) error {
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

func (m *trojanModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	settings, err := normalizeTrojanFlows(inbound.Settings)
	if err != nil {
		return nil, err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, settings, streamSettings), nil
}
