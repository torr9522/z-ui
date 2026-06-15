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

func (m *trojanModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: true,
		Fields: []FormField{
			{Name: "clients.0.password", Label: "密码", Type: "password", Required: true},
		},
	}
}

func (m *trojanModule) Validate(inbound *model.Inbound) error {
	settings, err := normalizeTrojanFlows(inbound.Settings)
	if err != nil {
		return err
	}
	inbound.Settings = settings
	decoded, err := decodeObject(settings)
	if err != nil {
		return err
	}
	if err := validateClients(decoded, "clients"); err != nil {
		return err
	}
	if err := validateClientPasswords(decoded, "clients"); err != nil {
		return err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return err
	}
	return validateRealityRequired(streamSettings)
}

func (m *trojanModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	settings, err := normalizeTrojanFlows(inbound.Settings)
	if err != nil {
		return nil, err
	}
	inbound.Settings = settings
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, settings, streamSettings), nil
}
