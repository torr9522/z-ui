package protocol

import (
	"errors"
	"strings"
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
	normalizeInboundProtocol(inbound, m.name)
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stringValue(settings["network"])) == "" {
		settings["network"] = "tcp,udp"
	}
	if _, exists := settings["followRedirect"]; !exists {
		settings["followRedirect"] = false
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return nil, err
	}
	clearTransportState(inbound)
	return inbound, nil
}

func (m *dokodemoModule) Validate(inbound *model.Inbound) error {
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
	network := strings.TrimSpace(strings.ToLower(stringValue(settings["network"])))
	if network == "" {
		settings["network"] = "tcp,udp"
	} else if network != "tcp,udp" && network != "tcp" && network != "udp" {
		return errors.New("dokodemo-door/tunnel network must be tcp, udp, or tcp,udp")
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return err
	}
	clearTransportState(inbound)
	return nil
}

func (m *dokodemoModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}
