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
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *vlessModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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

func (m *vlessModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	return normalizeInboundSchemaState(inbound, m.name, false)
}
