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
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *trojanModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	normalized, err := normalizeInboundSchemaState(inbound, m.name, true)
	if err != nil {
		return nil, err
	}
	schema, err := buildInboundSchema(normalized)
	if err != nil {
		return nil, err
	}
	return BuildInboundByProtocol(schema)
}
