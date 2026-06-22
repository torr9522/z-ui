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
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *mixedModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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
