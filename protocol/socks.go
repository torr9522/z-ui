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
	normalizeInboundProtocol(inbound, m.name)
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return nil, err
	}
	auth := stringValue(settings["auth"])
	if auth == "" {
		settings["auth"] = "noauth"
	}
	if stringValue(settings["ip"]) == "" {
		settings["ip"] = "127.0.0.1"
	}
	if _, exists := settings["udp"]; !exists {
		settings["udp"] = true
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return nil, err
	}
	clearTransportState(inbound)
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
		for _, item := range accounts {
			account, ok := item.(map[string]interface{})
			if !ok {
				return errors.New("socks account must be an object")
			}
			if stringValue(account["user"]) == "" {
				return errors.New("socks username must not be empty")
			}
			if stringValue(account["pass"]) == "" {
				return errors.New("socks password must not be empty")
			}
		}
	}
	clearTransportState(inbound)
	return nil
}

func (m *socksModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}
