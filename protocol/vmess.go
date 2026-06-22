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

func (m *vmessModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: true,
		Fields: []FormField{
			{Name: "clients.0.id", Label: "id", Type: "uuid", Required: true},
			{Name: "disableInsecureEncryption", Label: "禁用不安全加密", Type: "switch", Default: false},
		},
	}
}

func (m *vmessModule) Validate(inbound *model.Inbound) error {
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
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
