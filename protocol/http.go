package protocol

import (
	"errors"
	"x-ui/database/model"
	"x-ui/xray"
)

type httpModule struct {
	commonModule
}

func newHTTPModule() Module {
	return &httpModule{commonModule: commonModule{name: "http", supportsClients: false}}
}

func (m *httpModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: false,
		Fields: []FormField{
			{Name: "auth", Label: "密码认证", Type: "switch", Default: false},
			{Name: "accounts.0.user", Label: "用户名", Type: "text"},
			{Name: "accounts.0.pass", Label: "密码", Type: "password"},
		},
	}
}

func (m *httpModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	normalizeInboundProtocol(inbound, m.name)
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return nil, err
	}
	if _, exists := settings["allowTransparent"]; !exists {
		settings["allowTransparent"] = false
	}
	if _, exists := settings["auth"]; !exists {
		settings["auth"] = len(interfaceSlice(settings["accounts"])) > 0
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return nil, err
	}
	clearTransportState(inbound)
	return inbound, nil
}

func (m *httpModule) Validate(inbound *model.Inbound) error {
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
	if boolValue(settings["auth"]) {
		accounts, ok := settings["accounts"].([]interface{})
		if !ok || len(accounts) == 0 {
			return errors.New("http password auth requires at least one account")
		}
		for _, item := range accounts {
			account, ok := item.(map[string]interface{})
			if !ok {
				return errors.New("http account must be an object")
			}
			if stringValue(account["user"]) == "" {
				return errors.New("http username must not be empty")
			}
			if stringValue(account["pass"]) == "" {
				return errors.New("http password must not be empty")
			}
		}
	}
	clearTransportState(inbound)
	return nil
}

func (m *httpModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}
