package protocol

import (
	"errors"
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
			{Name: "auth", Label: "认证", Type: "select", Default: "password", Options: []FormOption{
				{Label: "password", Value: "password"},
				{Label: "noauth", Value: "noauth"},
			}},
			{Name: "accounts.0.user", Label: "用户名", Type: "text"},
			{Name: "accounts.0.pass", Label: "密码", Type: "password"},
			{Name: "udp", Label: "启用 UDP", Type: "switch", Default: false},
			{Name: "ip", Label: "IP", Type: "text", Default: "127.0.0.1"},
		},
	}
}

func (m *socksModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	normalizeInboundProtocol(inbound, m.name)
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return nil, err
	}
	auth := stringValue(settings["auth"])
	if auth == "" {
		settings["auth"] = "password"
	}
	if stringValue(settings["ip"]) == "" {
		settings["ip"] = "127.0.0.1"
	}
	if _, exists := settings["udp"]; !exists {
		settings["udp"] = false
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return nil, err
	}
	return inbound, nil
}

func (m *socksModule) Validate(inbound *model.Inbound) error {
	normalizeInboundProtocol(inbound, m.name)
	migrated, err := m.Migrate(inbound)
	if err != nil {
		return err
	}
	inbound = migrated
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return err
	}
	auth := stringValue(settings["auth"])
	switch auth {
	case "", "password", "noauth":
	default:
		return errors.New("socks auth must be password or noauth")
	}
	if auth == "password" {
		accounts, ok := settings["accounts"].([]interface{})
		if !ok || len(accounts) == 0 {
			return errors.New("socks password auth requires at least one account")
		}
	}
	if inbound.StreamSettings != "" && inbound.StreamSettings != "{}" {
		_, err = normalizeStreamSettings(m.name, inbound.StreamSettings)
	}
	return err
}

func (m *socksModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	streamSettings := inbound.StreamSettings
	if inbound.StreamSettings != "" && inbound.StreamSettings != "{}" {
		normalized, err := normalizeStreamSettings(m.name, inbound.StreamSettings)
		if err != nil {
			return nil, err
		}
		streamSettings = normalized
	}
	return buildInboundConfig(inbound, inbound.Settings, streamSettings), nil
}
