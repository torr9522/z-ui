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
