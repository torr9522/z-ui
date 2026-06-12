package protocol

import (
	"errors"
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
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return err
	}
	if stringValue(settings["password"]) == "" {
		if clients, ok := settings["clients"]; !ok || clients == nil {
			return errors.New("shadowsocks requires password or clients")
		}
	}
	_, err = normalizeStreamSettings(m.name, inbound.StreamSettings)
	return err
}

func (m *shadowsocksModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	streamSettings, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, streamSettings), nil
}
