package protocol

import (
	"fmt"
	"strings"

	"github.com/xtls/xray-core/common/uuid"
)

var shadowsocksMethods = map[string]struct{}{
	"chacha20-poly1305": {},
	"aes-256-gcm":       {},
	"aes-128-gcm":       {},
}

var shadowsocksNetworks = map[string]struct{}{
	"tcp,udp": {},
	"tcp":     {},
	"udp":     {},
}

func ValidateInboundSchema(schema *InboundSchema) error {
	if schema == nil {
		return fmt.Errorf("schema must not be nil")
	}
	switch schema.Protocol {
	case "vmess":
		return validateVMessSchema(schema)
	case "vless":
		return validateVLESSSchema(schema)
	case "trojan":
		return validateTrojanSchema(schema)
	case "shadowsocks":
		return validateShadowsocksSchema(schema)
	case "socks":
		return validateSocksSchema(schema)
	case "http":
		return validateHTTPSchema(schema)
	case "mixed":
		return validateMixedSchema(schema)
	case "dokodemo-door", "tunnel":
		return validateDokodemoSchema(schema)
	default:
		return fmt.Errorf("unsupported protocol %q", schema.Protocol)
	}
}

func validateVMessSchema(schema *InboundSchema) error {
	if err := validateSingleClientUUID(schema.Clients); err != nil {
		return err
	}
	return validateStreamCompatibility(schema.Protocol, schema.StreamSettings, false)
}

func validateVLESSSchema(schema *InboundSchema) error {
	if err := validateSingleClientUUID(schema.Clients); err != nil {
		return err
	}
	if strings.TrimSpace(stringValue(schema.Settings["decryption"])) == "" {
		return fmt.Errorf("settings.decryption must not be empty")
	}
	if err := validateStreamCompatibility(schema.Protocol, schema.StreamSettings, true); err != nil {
		return err
	}
	return validateRealityRequiredInStream(schema.StreamSettings)
}

func validateTrojanSchema(schema *InboundSchema) error {
	if len(schema.Clients) == 0 {
		return fmt.Errorf("clients must contain at least one client")
	}
	password := strings.TrimSpace(stringValue(schema.Clients[0]["password"]))
	if password == "" {
		return fmt.Errorf("clients[0].password must not be empty")
	}
	if err := validateStreamCompatibility(schema.Protocol, schema.StreamSettings, true); err != nil {
		return err
	}
	return validateRealityRequiredInStream(schema.StreamSettings)
}

func validateShadowsocksSchema(schema *InboundSchema) error {
	method := strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["method"])))
	if _, ok := shadowsocksMethods[method]; !ok {
		return fmt.Errorf("settings.method must be a supported shadowsocks method")
	}
	password := strings.TrimSpace(stringValue(schema.Settings["password"]))
	if password == "" {
		return fmt.Errorf("settings.password must not be empty")
	}
	network := strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["network"])))
	if _, ok := shadowsocksNetworks[network]; !ok {
		return fmt.Errorf("settings.network must be tcp, udp, or tcp,udp")
	}
	return validateStreamCompatibility(schema.Protocol, schema.StreamSettings, false)
}

func validateSocksSchema(schema *InboundSchema) error {
	auth := strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["auth"])))
	switch auth {
	case "", "noauth":
		return nil
	case "password":
		accounts, ok := schema.Settings["accounts"].([]interface{})
		if !ok || len(accounts) == 0 {
			return fmt.Errorf("socks password auth requires at least one account")
		}
		for _, item := range accounts {
			account, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("socks account must be an object")
			}
			if strings.TrimSpace(stringValue(account["user"])) == "" {
				return fmt.Errorf("socks username must not be empty")
			}
			if strings.TrimSpace(stringValue(account["pass"])) == "" {
				return fmt.Errorf("socks password must not be empty")
			}
		}
		return nil
	default:
		return fmt.Errorf("socks auth must be password or noauth")
	}
}

func validateHTTPSchema(schema *InboundSchema) error {
	authEnabled := boolValue(schema.Settings["auth"])
	if !authEnabled {
		return validateHTTPStreamSettingsSchema(schema.StreamSettings)
	}
	accounts, ok := schema.Settings["accounts"].([]interface{})
	if !ok || len(accounts) == 0 {
		return fmt.Errorf("http password auth requires at least one account")
	}
	for _, item := range accounts {
		account, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("http account must be an object")
		}
		if strings.TrimSpace(stringValue(account["user"])) == "" {
			return fmt.Errorf("http username must not be empty")
		}
		if strings.TrimSpace(stringValue(account["pass"])) == "" {
			return fmt.Errorf("http password must not be empty")
		}
	}
	return validateHTTPStreamSettingsSchema(schema.StreamSettings)
}

func validateMixedSchema(schema *InboundSchema) error {
	if len(schema.Clients) > 0 {
		return fmt.Errorf("mixed must not contain clients")
	}
	return nil
}

func validateDokodemoSchema(schema *InboundSchema) error {
	network := strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["network"])))
	switch network {
	case "", "tcp,udp", "tcp", "udp":
		return nil
	default:
		return fmt.Errorf("dokodemo-door/tunnel network must be tcp, udp, or tcp,udp")
	}
}

func validateSingleClientUUID(clients []map[string]interface{}) error {
	if len(clients) == 0 {
		return fmt.Errorf("clients must contain at least one client")
	}
	id := strings.TrimSpace(stringValue(clients[0]["id"]))
	if id == "" {
		return fmt.Errorf("clients[0].id must not be empty")
	}
	if !strictUUIDPattern.MatchString(id) {
		return fmt.Errorf("clients[0].id must be a valid UUID")
	}
	if _, err := uuid.ParseString(id); err != nil {
		return fmt.Errorf("clients[0].id must be a valid UUID")
	}
	return nil
}

func validateStreamCompatibility(protocolName string, stream map[string]interface{}, allowReality bool) error {
	security := strings.TrimSpace(strings.ToLower(stringValue(stream["security"])))
	if security == "null" {
		security = ""
	}
	network := strings.TrimSpace(strings.ToLower(stringValue(stream["network"])))
	if network == "null" {
		network = ""
	}
	if security == "" || security == "none" {
		return nil
	}
	if security == "tls" {
		return nil
	}
	if security == "reality" {
		if !allowReality {
			return fmt.Errorf("%s does not support reality", protocolName)
		}
		switch network {
		case "tcp", "xhttp", "grpc":
			return nil
		default:
			return fmt.Errorf("reality only supports tcp, xhttp, and grpc")
		}
	}
	return fmt.Errorf("unsupported stream security %q", security)
}

func validateRealityRequiredInStream(stream map[string]interface{}) error {
	if strings.ToLower(stringValue(stream["security"])) != "reality" {
		return nil
	}
	reality, ok := stream["realitySettings"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("reality requires realitySettings")
	}
	if strings.TrimSpace(stringValue(reality["privateKey"])) == "" {
		return fmt.Errorf("reality privateKey must not be empty")
	}
	if strings.TrimSpace(stringValue(reality["dest"])) == "" {
		return fmt.Errorf("reality dest must not be empty")
	}
	return nil
}

func validateHTTPStreamSettingsSchema(stream map[string]interface{}) error {
	security := strings.TrimSpace(strings.ToLower(stringValue(stream["security"])))
	if security == "null" {
		security = ""
	}
	switch security {
	case "", "none":
		return nil
	case "tls":
		tlsSettings, ok := stream["tlsSettings"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("http tls requires tlsSettings")
		}
		certificates, ok := tlsSettings["certificates"].([]interface{})
		if !ok || len(certificates) == 0 {
			return fmt.Errorf("http tls requires certificate")
		}
		return nil
	default:
		return fmt.Errorf("http security must be none or tls")
	}
}
