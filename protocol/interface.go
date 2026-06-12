package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type FormSchema struct {
	Protocol        string
	SupportsClients bool
	Fields          []FormField
}

type FormField struct {
	Name     string
	Type     string
	Required bool
}

type Module interface {
	Name() string
	BuildInbound(*model.Inbound) (*xray.InboundConfig, error)
	Validate(*model.Inbound) error
	Migrate(*model.Inbound) (*model.Inbound, error)
	FormSchema() FormSchema
}
