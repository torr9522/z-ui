package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type socksModule struct {
	commonModule
}

func newSocksModule() Module {
	return &socksModule{commonModule: commonModule{name: "socks", supportsClients: false}}
}

func (m *socksModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: false,
		Fields: []FormField{
			{Name: "auth", Label: "认证", Type: "select", Default: "noauth", Options: []FormOption{
				{Label: "password", Value: "password"},
				{Label: "noauth", Value: "noauth"},
			}},
			{Name: "accounts.0.user", Label: "用户名", Type: "text"},
			{Name: "accounts.0.pass", Label: "密码", Type: "password"},
			{Name: "udp", Label: "启用 UDP", Type: "switch", Default: true},
			{Name: "ip", Label: "IP", Type: "text", Default: "127.0.0.1"},
		},
	}
}

func (m *socksModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	return normalizeInboundSchemaState(inbound, m.name, false)
}

func (m *socksModule) Validate(inbound *model.Inbound) error {
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *socksModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}
