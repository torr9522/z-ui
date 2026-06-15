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

func (m *vlessModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: true,
		Fields: []FormField{
			{Name: "clients.0.id", Label: "id", Type: "uuid", Required: true},
			{Name: "clients.0.flow", Label: "flow", Type: "select", Default: "", Options: []FormOption{
				{Label: "无", Value: ""},
				{Label: "xtls-rprx-vision", Value: "xtls-rprx-vision"},
			}},
			{Name: "decryption", Label: "decryption", Type: "hidden", Default: "none"},
		},
	}
}

func (m *vlessModule) Validate(inbound *model.Inbound) error {
	settings, err := normalizeVLESSSettings(inbound.Settings)
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
	if err := validateClientUUIDs(decoded, "clients"); err != nil {
		return err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return err
	}
	return validateRealityRequired(streamSettings)
}

func (m *vlessModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	settings, err := normalizeVLESSSettings(inbound.Settings)
	if err != nil {
		return nil, err
	}
	settings, err = normalizeVisionFlows(settings)
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

func (m *vlessModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	settings, err := normalizeVLESSSettings(inbound.Settings)
	if err != nil {
		return nil, err
	}
	inbound.Settings = settings
	return inbound, nil
}
