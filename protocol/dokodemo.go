package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type dokodemoModule struct {
	commonModule
}

func newDokodemoModule(name string) Module {
	return &dokodemoModule{commonModule: commonModule{name: name, supportsClients: false}}
}

func (m *dokodemoModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: false,
		Fields: []FormField{
			{Name: "address", Label: "目标地址", Type: "text"},
			{Name: "port", Label: "目标端口", Type: "number"},
			{Name: "network", Label: "网络", Type: "select", Default: "tcp,udp", Options: []FormOption{
				{Label: "tcp+udp", Value: "tcp,udp"},
				{Label: "tcp", Value: "tcp"},
				{Label: "udp", Value: "udp"},
			}},
			{Name: "followRedirect", Label: "followRedirect", Type: "switch", Default: false},
		},
	}
}

func (m *dokodemoModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	return normalizeInboundSchemaState(inbound, m.name, false)
}

func (m *dokodemoModule) Validate(inbound *model.Inbound) error {
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	return err
}

func (m *dokodemoModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
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
