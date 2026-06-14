package protocol

import (
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
			{Name: "accounts.0.user", Label: "用户名", Type: "text"},
			{Name: "accounts.0.pass", Label: "密码", Type: "password"},
			{Name: "allowTransparent", Label: "allowTransparent", Type: "switch", Default: false},
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
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return nil, err
	}
	return inbound, nil
}

func (m *httpModule) Validate(inbound *model.Inbound) error {
	normalizeInboundProtocol(inbound, m.name)
	migrated, err := m.Migrate(inbound)
	if err != nil {
		return err
	}
	inbound = migrated
	if _, err := decodeObject(inbound.Settings); err != nil {
		return err
	}
	if inbound.StreamSettings != "" && inbound.StreamSettings != "{}" {
		_, err = normalizeStreamSettings(m.name, inbound.StreamSettings)
	}
	return err
}

func (m *httpModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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
