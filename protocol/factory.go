package protocol

import (
	"fmt"
	"os"
	"strings"
	"x-ui/database/model"
	"x-ui/xray"
)

func BuildInboundByProtocol(schema *InboundSchema) (*xray.InboundConfig, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema must not be nil")
	}
	switch strings.ToLower(schema.Protocol) {
	case "vmess":
		return buildVMessInbound(schema)
	case "vless":
		return buildVLESSInbound(schema)
	case "trojan":
		return buildTrojanInbound(schema)
	case "shadowsocks":
		return buildShadowsocksInbound(schema)
	case "socks":
		return buildSocksInbound(schema)
	case "http":
		return buildHTTPInbound(schema)
	case "mixed":
		return buildMixedInbound(schema)
	case "dokodemo-door", "tunnel":
		return BuildDokodemoInbound(schema)
	default:
		return nil, fmt.Errorf("unsupported protocol %q", schema.Protocol)
	}
}

func BuildDokodemoInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"address":        strings.TrimSpace(stringValue(schema.Settings["address"])),
		"port":           numericValue(schema.Settings["port"]),
		"network":        strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["network"]))),
		"followRedirect": boolValue(schema.Settings["followRedirect"]),
	}
	if settings["network"] == "" {
		settings["network"] = "tcp,udp"
	}
	return schemaInboundConfig(schema, settings, map[string]interface{}{})
}

func buildVMessInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"disableInsecureEncryption": boolValue(schema.Settings["disableInsecureEncryption"]),
	}
	if len(schema.Clients) > 0 {
		settings["clients"] = []interface{}{map[string]interface{}{
			"id":      strings.TrimSpace(stringValue(schema.Clients[0]["id"])),
			"alterId": 0,
			"email":   strings.TrimSpace(stringValue(schema.Clients[0]["email"])),
		}}
	}
	return schemaInboundConfig(schema, settings, schema.StreamSettings)
}

func buildVLESSInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"decryption": firstNonEmptyString(schema.Settings["decryption"], "none"),
	}
	if len(schema.Clients) > 0 {
		client := map[string]interface{}{
			"id": strings.TrimSpace(stringValue(schema.Clients[0]["id"])),
		}
		if flow := strings.TrimSpace(stringValue(schema.Clients[0]["flow"])); flow != "" {
			client["flow"] = flow
		}
		if email := strings.TrimSpace(stringValue(schema.Clients[0]["email"])); email != "" {
			client["email"] = email
		}
		settings["clients"] = []interface{}{client}
	}
	return schemaInboundConfig(schema, settings, schema.StreamSettings)
}

func buildTrojanInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{}
	if len(schema.Clients) > 0 {
		client := map[string]interface{}{
			"password": strings.TrimSpace(stringValue(schema.Clients[0]["password"])),
		}
		if email := strings.TrimSpace(stringValue(schema.Clients[0]["email"])); email != "" {
			client["email"] = email
		}
		settings["clients"] = []interface{}{client}
	}
	return schemaInboundConfig(schema, settings, schema.StreamSettings)
}

func buildShadowsocksInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"method":   strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["method"]))),
		"password": strings.TrimSpace(stringValue(schema.Settings["password"])),
		"network":  strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["network"]))),
	}
	if settings["method"] == "" {
		settings["method"] = "aes-256-gcm"
	}
	if settings["network"] == "" {
		settings["network"] = "tcp,udp"
	}
	return schemaInboundConfig(schema, settings, map[string]interface{}{})
}

func buildSocksInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"auth": strings.TrimSpace(strings.ToLower(stringValue(schema.Settings["auth"]))),
		"udp":  boolValueOrDefault(schema.Settings["udp"], true),
		"ip":   firstNonEmptyString(schema.Settings["ip"], "127.0.0.1"),
	}
	if accounts := filterAccounts(schema.Settings["accounts"]); len(accounts) > 0 && settings["auth"] == "password" {
		settings["accounts"] = accounts
	}
	if settings["auth"] == "" {
		settings["auth"] = "noauth"
	}
	return schemaInboundConfig(schema, settings, map[string]interface{}{})
}

func buildHTTPInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	settings := map[string]interface{}{
		"allowTransparent": boolValue(schema.Settings["allowTransparent"]),
	}
	if boolValue(schema.Settings["auth"]) {
		if accounts := filterAccounts(schema.Settings["accounts"]); len(accounts) > 0 {
			settings["accounts"] = accounts
		}
	}
	stream := cloneMap(schema.StreamSettings)
	if strings.ToLower(stringValue(stream["security"])) == "tls" {
		tlsSettings, _ := stream["tlsSettings"].(map[string]interface{})
		certificates := interfaceSlice(tlsSettings["certificates"])
		if len(certificates) == 0 {
			return nil, fmt.Errorf("http tls requires certificate")
		}
		certificate, _ := certificates[0].(map[string]interface{})
		certFile := strings.TrimSpace(stringValue(certificate["certificateFile"]))
		keyFile := strings.TrimSpace(stringValue(certificate["keyFile"]))
		if certFile == "" {
			return nil, fmt.Errorf("http tls certificateFile must not be empty")
		}
		if keyFile == "" {
			return nil, fmt.Errorf("http tls keyFile must not be empty")
		}
		if _, err := os.Stat(certFile); err != nil {
			return nil, fmt.Errorf("http tls certificateFile does not exist")
		}
		if _, err := os.Stat(keyFile); err != nil {
			return nil, fmt.Errorf("http tls keyFile does not exist")
		}
	}
	return schemaInboundConfig(schema, settings, stream)
}

func buildMixedInbound(schema *InboundSchema) (*xray.InboundConfig, error) {
	return schemaInboundConfig(schema, map[string]interface{}{}, map[string]interface{}{})
}

func schemaInboundConfig(schema *InboundSchema, settings map[string]interface{}, streamSettings map[string]interface{}) (*xray.InboundConfig, error) {
	rawSettings, err := encodeObject(settings)
	if err != nil {
		return nil, err
	}
	rawStreamSettings, err := encodeObject(streamSettings)
	if err != nil {
		return nil, err
	}
	inbound := &model.Inbound{
		Listen:         schema.Listen,
		Port:           schema.Port,
		Protocol:       model.Protocol(schema.Protocol),
		Settings:       rawSettings,
		StreamSettings: rawStreamSettings,
		Tag:            schema.Tag,
		Sniffing:       emptyObject(),
	}
	return buildInboundConfig(inbound, rawSettings, rawStreamSettings), nil
}
