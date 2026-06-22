package protocol

import (
	"errors"
	"os"
	"strings"
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
	return normalizeInboundSchemaState(inbound, m.name, false)
}

func (m *httpModule) Validate(inbound *model.Inbound) error {
	_, err := normalizeInboundSchemaState(inbound, m.name, true)
	if err != nil {
		return err
	}
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return err
	}
	authEnabled := boolValue(settings["auth"])
	delete(settings, "auth")
	if !authEnabled {
		delete(settings, "accounts")
	}
	inbound.Settings, err = encodeObject(settings)
	if err != nil {
		return err
	}
	if strings.TrimSpace(inbound.StreamSettings) == "" {
		inbound.StreamSettings = emptyObject()
		return nil
	}
	stream, err := normalizeHTTPStreamSettings(inbound.StreamSettings)
	if err != nil {
		return err
	}
	inbound.StreamSettings = stream
	return nil
}

func (m *httpModule) BuildInbound(inbound *model.Inbound) (*xray.InboundConfig, error) {
	if err := m.Validate(inbound); err != nil {
		return nil, err
	}
	return buildInboundConfig(inbound, inbound.Settings, inbound.StreamSettings), nil
}

func normalizeHTTPStreamSettings(raw string) (string, error) {
	stream, err := decodeObject(raw)
	if err != nil {
		return "", err
	}
	securityValue, exists := stream["security"]
	security := strings.ToLower(strings.TrimSpace(stringValue(securityValue)))
	if !exists || security == "" || security == "null" {
		security = "none"
	}
	switch security {
	case "none":
		return emptyObject(), nil
	case "tls":
		tlsSettings, ok := stream["tlsSettings"].(map[string]interface{})
		if !ok {
			return "", errors.New("http tls requires tlsSettings")
		}
		certificates, ok := tlsSettings["certificates"].([]interface{})
		if !ok || len(certificates) == 0 {
			return "", errors.New("http tls requires certificate")
		}
		certificate, ok := certificates[0].(map[string]interface{})
		if !ok {
			return "", errors.New("http tls certificate must be an object")
		}
		certFile := strings.TrimSpace(stringValue(certificate["certificateFile"]))
		keyFile := strings.TrimSpace(stringValue(certificate["keyFile"]))
		if certFile == "" {
			return "", errors.New("http tls certificateFile must not be empty")
		}
		if keyFile == "" {
			return "", errors.New("http tls keyFile must not be empty")
		}
		if _, err := os.Stat(certFile); err != nil {
			return "", errors.New("http tls certificateFile does not exist")
		}
		if _, err := os.Stat(keyFile); err != nil {
			return "", errors.New("http tls keyFile does not exist")
		}
		delete(stream, "network")
		delete(stream, "realitySettings")
		delete(stream, "tcpSettings")
		delete(stream, "kcpSettings")
		delete(stream, "wsSettings")
		delete(stream, "httpSettings")
		delete(stream, "xhttpSettings")
		delete(stream, "quicSettings")
		delete(stream, "grpcSettings")
		stream["security"] = "tls"
		stream["tlsSettings"] = tlsSettings
		return encodeObject(stream)
	default:
		return "", errors.New("http security must be none or tls")
	}
}
