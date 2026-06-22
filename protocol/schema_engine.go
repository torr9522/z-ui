package protocol

import (
	"strings"
	"x-ui/database/model"
)

type InboundSchema struct {
	Protocol       string
	Clients        []map[string]interface{}
	Settings       map[string]interface{}
	StreamSettings map[string]interface{}
}

func normalizeInboundSchemaState(inbound *model.Inbound, protocolName string, validate bool) (*model.Inbound, error) {
	normalizeInboundProtocol(inbound, protocolName)

	schema, err := buildInboundSchema(inbound)
	if err != nil {
		return nil, err
	}
	schema.Protocol = protocolName
	schema.normalize()
	if validate {
		if err := ValidateInboundSchema(schema); err != nil {
			return nil, err
		}
	}
	if err := schema.apply(inbound); err != nil {
		return nil, err
	}
	return inbound, nil
}

func buildInboundSchema(inbound *model.Inbound) (*InboundSchema, error) {
	settings, err := decodeObject(inbound.Settings)
	if err != nil {
		return nil, err
	}
	streamSettings, err := decodeObject(inbound.StreamSettings)
	if err != nil {
		return nil, err
	}
	clients := schemaClients(settings)
	delete(settings, "clients")

	if len(settings) == 0 {
		settings = defaultSchemaSettings(strings.ToLower(string(inbound.Protocol)))
	}

	return &InboundSchema{
		Protocol:       strings.ToLower(string(inbound.Protocol)),
		Clients:        clients,
		Settings:       settings,
		StreamSettings: streamSettings,
	}, nil
}

func (s *InboundSchema) apply(inbound *model.Inbound) error {
	settings := cloneMap(s.Settings)
	if len(s.Clients) > 0 {
		clients := make([]interface{}, 0, len(s.Clients))
		for _, client := range s.Clients {
			clients = append(clients, cloneMap(client))
		}
		settings["clients"] = clients
	}

	rawSettings, err := encodeObject(settings)
	if err != nil {
		return err
	}
	rawStreamSettings, err := encodeObject(cloneMap(s.StreamSettings))
	if err != nil {
		return err
	}

	inbound.Protocol = model.Protocol(s.Protocol)
	inbound.Settings = rawSettings
	inbound.StreamSettings = rawStreamSettings

	switch s.Protocol {
	case "mixed", "socks", "http", "dokodemo-door", "tunnel":
		clearTransportState(inbound)
		inbound.StreamSettings = rawStreamSettings
	}

	return nil
}

func (s *InboundSchema) normalize() {
	if s.Settings == nil {
		s.Settings = map[string]interface{}{}
	}
	if s.StreamSettings == nil {
		s.StreamSettings = map[string]interface{}{}
	}

	switch s.Protocol {
	case "vmess":
		s.normalizeVMess()
	case "vless":
		s.normalizeVLESS()
	case "trojan":
		s.normalizeTrojan()
	case "shadowsocks":
		s.normalizeShadowsocks()
	case "socks":
		s.normalizeSocks()
	case "http":
		s.normalizeHTTP()
	case "mixed":
		s.Clients = nil
		s.StreamSettings = map[string]interface{}{}
	case "dokodemo-door", "tunnel":
		s.normalizeDokodemo()
	}
}

func (s *InboundSchema) normalizeVMess() {
	s.Clients = singleClientOnly(s.Clients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["id"] = strings.TrimSpace(stringValue(client["id"]))
		client["alterId"] = 0
		delete(client, "flow")
	}
	delete(s.Settings, "decryption")
	s.Settings["disableInsecureEncryption"] = boolValue(s.Settings["disableInsecureEncryption"])
	s.normalizeTransport(false)
}

func (s *InboundSchema) normalizeVLESS() {
	s.Clients = singleClientOnly(s.Clients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["id"] = strings.TrimSpace(stringValue(client["id"]))
		normalizeClientFlow(client)
	}
	if strings.TrimSpace(stringValue(s.Settings["decryption"])) == "" {
		s.Settings["decryption"] = "none"
	}
	s.normalizeTransport(true)
}

func (s *InboundSchema) normalizeTrojan() {
	s.Clients = singleClientOnly(s.Clients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["password"] = strings.TrimSpace(stringValue(client["password"]))
		delete(client, "flow")
	}
	s.normalizeTransport(true)
}

func (s *InboundSchema) normalizeShadowsocks() {
	s.Clients = singleClientOnly(s.Clients)
	if strings.TrimSpace(stringValue(s.Settings["password"])) == "" && len(s.Clients) > 0 {
		s.Settings["password"] = strings.TrimSpace(stringValue(s.Clients[0]["password"]))
	}
	if strings.TrimSpace(stringValue(s.Settings["method"])) == "" {
		s.Settings["method"] = "aes-256-gcm"
	}
	if strings.TrimSpace(stringValue(s.Settings["network"])) == "" {
		s.Settings["network"] = "tcp,udp"
	}
	s.Clients = nil
	s.normalizeTransport(false)
}

func (s *InboundSchema) normalizeSocks() {
	s.Clients = nil
	auth := strings.TrimSpace(strings.ToLower(stringValue(s.Settings["auth"])))
	switch auth {
	case "", "noauth":
		s.Settings["auth"] = "noauth"
		delete(s.Settings, "accounts")
	case "password":
		s.Settings["auth"] = "password"
	default:
		s.Settings["auth"] = auth
	}
	if strings.TrimSpace(stringValue(s.Settings["ip"])) == "" {
		s.Settings["ip"] = "127.0.0.1"
	}
	if _, exists := s.Settings["udp"]; !exists {
		s.Settings["udp"] = true
	} else {
		s.Settings["udp"] = boolValue(s.Settings["udp"])
	}
	s.StreamSettings = map[string]interface{}{}
}

func (s *InboundSchema) normalizeHTTP() {
	s.Clients = nil
	if _, exists := s.Settings["allowTransparent"]; !exists {
		s.Settings["allowTransparent"] = false
	} else {
		s.Settings["allowTransparent"] = boolValue(s.Settings["allowTransparent"])
	}
	authEnabled := boolValue(s.Settings["auth"])
	if !authEnabled {
		delete(s.Settings, "accounts")
	}
	s.Settings["auth"] = authEnabled
}

func (s *InboundSchema) normalizeDokodemo() {
	s.Clients = nil
	if strings.TrimSpace(stringValue(s.Settings["network"])) == "" {
		s.Settings["network"] = "tcp,udp"
	}
	if _, exists := s.Settings["followRedirect"]; !exists {
		s.Settings["followRedirect"] = false
	} else {
		s.Settings["followRedirect"] = boolValue(s.Settings["followRedirect"])
	}
	s.StreamSettings = map[string]interface{}{}
}

func (s *InboundSchema) normalizeTransport(allowReality bool) {
	if s.StreamSettings == nil {
		s.StreamSettings = map[string]interface{}{}
	}
	if len(s.StreamSettings) == 0 {
		return
	}

	network := strings.ToLower(strings.TrimSpace(stringValue(s.StreamSettings["network"])))
	switch network {
	case "splithttp", "http":
		s.StreamSettings["network"] = "xhttp"
	case "":
	default:
		s.StreamSettings["network"] = network
	}

	security := strings.ToLower(strings.TrimSpace(stringValue(s.StreamSettings["security"])))
	if security == "xtls" {
		security = "tls"
		if _, exists := s.StreamSettings["tlsSettings"]; !exists {
			if legacy, ok := s.StreamSettings["xtlsSettings"]; ok {
				s.StreamSettings["tlsSettings"] = legacy
			}
		}
	}
	delete(s.StreamSettings, "xtlsSettings")

	if security == "reality" && !allowReality {
		security = "none"
		delete(s.StreamSettings, "realitySettings")
	}

	if security != "tls" && security != "reality" {
		security = "none"
	}
	s.StreamSettings["security"] = security
}

func schemaClients(settings map[string]interface{}) []map[string]interface{} {
	rawClients, ok := settings["clients"].([]interface{})
	if !ok || len(rawClients) == 0 {
		return nil
	}
	clients := make([]map[string]interface{}, 0, len(rawClients))
	for _, item := range rawClients {
		client, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		clients = append(clients, cloneMap(client))
	}
	return clients
}

func singleClientOnly(clients []map[string]interface{}) []map[string]interface{} {
	if len(clients) == 0 {
		return nil
	}
	return []map[string]interface{}{cloneMap(clients[0])}
}

func schemaAllowsVision(stream map[string]interface{}) bool {
	security := strings.ToLower(strings.TrimSpace(stringValue(stream["security"])))
	return security == "tls" || security == "reality"
}

func normalizeClientFlow(client map[string]interface{}) {
	flow := strings.ToLower(strings.TrimSpace(stringValue(client["flow"])))
	switch flow {
	case "", "xtls-rprx-vision":
		if flow == "" {
			delete(client, "flow")
		} else {
			client["flow"] = flow
		}
	case "xtls-rprx-origin", "xtls-rprx-direct":
		client["flow"] = "xtls-rprx-vision"
	default:
		delete(client, "flow")
	}
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func defaultSchemaSettings(protocolName string) map[string]interface{} {
	switch protocolName {
	case "vmess":
		return map[string]interface{}{
			"disableInsecureEncryption": false,
		}
	case "vless":
		return map[string]interface{}{
			"decryption": "none",
		}
	case "trojan":
		return map[string]interface{}{}
	case "shadowsocks":
		return map[string]interface{}{
			"method":  "aes-256-gcm",
			"network": "tcp,udp",
		}
	case "socks":
		return map[string]interface{}{
			"auth": "noauth",
			"udp":  true,
			"ip":   "127.0.0.1",
		}
	case "http":
		return map[string]interface{}{
			"auth":             false,
			"allowTransparent": false,
		}
	case "dokodemo-door", "tunnel":
		return map[string]interface{}{
			"network":        "tcp,udp",
			"followRedirect": false,
		}
	default:
		return map[string]interface{}{}
	}
}
