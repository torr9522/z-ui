package protocol

import (
	"x-ui/database/model"
	"x-ui/xray"
)

type FormSchema struct {
	Protocol        string      `json:"protocol"`
	SupportsClients bool        `json:"supportsClients"`
	Fields          []FormField `json:"fields"`
}

type FormField struct {
	Name     string       `json:"name"`
	Label    string       `json:"label"`
	Type     string       `json:"type"`
	Required bool         `json:"required"`
	Default  interface{}  `json:"default,omitempty"`
	Options  []FormOption `json:"options,omitempty"`
}

type FormOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

type Module interface {
	Name() string
	BuildInbound(*model.Inbound) (*xray.InboundConfig, error)
	Validate(*model.Inbound) error
	Migrate(*model.Inbound) (*model.Inbound, error)
	FormSchema() FormSchema
}
