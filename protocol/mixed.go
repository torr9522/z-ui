package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type mixedModule struct {
	commonModule
}

func newMixedModule() Module {
	return &mixedModule{commonModule: commonModule{name: "mixed", supportsClients: false}}
}

func (m *mixedModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: false,
		Fields:          []FormField{},
	}
}

func (m *mixedModule) Validate(inbound *model.Inbound) error {
	clearTransportState(inbound)
	_, err := decodeObject(inbound.Settings)
	return err
}

func (m *mixedModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}
