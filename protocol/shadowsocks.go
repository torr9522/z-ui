package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type shadowsocksModule struct {
	commonModule
}

func newShadowsocksModule() Module {
	return &shadowsocksModule{commonModule: commonModule{name: "shadowsocks", supportsClients: true}}
}

func (m *shadowsocksModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: true,
		Fields: []FormField{
			{Name: "method", Label: "加密", Type: "select", Required: true, Default: "aes-256-gcm", Options: []FormOption{
				{Label: "chacha20-poly1305", Value: "chacha20-poly1305"},
				{Label: "aes-256-gcm", Value: "aes-256-gcm"},
				{Label: "aes-128-gcm", Value: "aes-128-gcm"},
			}},
			{Name: "password", Label: "密码", Type: "password", Required: true},
			{Name: "network", Label: "网络", Type: "select", Default: "tcp,udp", Options: []FormOption{
				{Label: "tcp+udp", Value: "tcp,udp"},
				{Label: "tcp", Value: "tcp"},
				{Label: "udp", Value: "udp"},
			}},
		},
	}
}

func (m *shadowsocksModule) Validate(inbound *model.Inbound) error {
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *shadowsocksModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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
