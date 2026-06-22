package protocol

import (
	"strings"
	"x-ui/database/model"
)

type InboundSchema struct {
	Listen         string
	Port           int
	Tag            string
	Sniffing       map[string]interface{}
	Protocol       string
	Clients        []map[string]interface{}
	Settings       map[string]interface{}
	StreamSettings map[string]interface{}
}

func BuildSchemaFromInbound(inbound *model.Inbound, protocolName string, validate bool) (*InboundSchema, error) {
	normalized, err := normalizeInboundSchemaState(inbound, protocolName, validate)
	if err != nil {
		return nil, err
	}
	return buildInboundSchema(normalized)
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
		Listen:         inbound.Listen,
		Port:           inbound.Port,
		Tag:            inbound.Tag,
		Sniffing:       map[string]interface{}{},
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
	inbound.Sniffing = emptyObject()

	return nil
}

func (s *InboundSchema) normalize() {
	rawSettings := cloneMap(s.Settings)
	rawStreamSettings := cloneMap(s.StreamSettings)
	rawClients := cloneClients(s.Clients)

	ResetInboundState(s)

	switch s.Protocol {
	case "vmess":
		s.normalizeVMess(rawSettings, rawClients, rawStreamSettings)
	case "vless":
		s.normalizeVLESS(rawSettings, rawClients, rawStreamSettings)
	case "trojan":
		s.normalizeTrojan(rawSettings, rawClients, rawStreamSettings)
	case "shadowsocks":
		s.normalizeShadowsocks(rawSettings, rawClients)
	case "socks":
		s.normalizeSocks(rawSettings)
	case "http":
		s.normalizeHTTP(rawSettings, rawStreamSettings)
	case "mixed":
		s.Settings = map[string]interface{}{}
		s.StreamSettings = map[string]interface{}{}
	case "dokodemo-door", "tunnel":
		s.normalizeDokodemo(rawSettings)
	}
}

func ResetInboundState(schema *InboundSchema) {
	schema.Clients = nil
	schema.Settings = defaultSchemaSettings(schema.Protocol)
	schema.StreamSettings = map[string]interface{}{}
	schema.Sniffing = map[string]interface{}{}
}

func (s *InboundSchema) normalizeVMess(rawSettings map[string]interface{}, rawClients []map[string]interface{}, rawStream map[string]interface{}) {
	s.Clients = singleClientOnly(rawClients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["id"] = strings.TrimSpace(stringValue(client["id"]))
		client["alterId"] = 0
		delete(client, "flow")
		delete(client, "password")
	}
	s.Settings["disableInsecureEncryption"] = boolValue(rawSettings["disableInsecureEncryption"])
	s.StreamSettings = rebuildTransportSettings(rawStream, transportOptions{
		allowTLS:     true,
		allowReality: false,
	})
}

func (s *InboundSchema) normalizeVLESS(rawSettings map[string]interface{}, rawClients []map[string]interface{}, rawStream map[string]interface{}) {
	s.Clients = singleClientOnly(rawClients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["id"] = strings.TrimSpace(stringValue(client["id"]))
		normalizeClientFlow(client)
	}
	decryption := strings.TrimSpace(stringValue(rawSettings["decryption"]))
	if decryption == "" {
		decryption = "none"
	}
	s.Settings["decryption"] = decryption
	s.StreamSettings = rebuildTransportSettings(rawStream, transportOptions{
		allowTLS:     true,
		allowReality: true,
	})
}

func (s *InboundSchema) normalizeTrojan(rawSettings map[string]interface{}, rawClients []map[string]interface{}, rawStream map[string]interface{}) {
	s.Clients = singleClientOnly(rawClients)
	if len(s.Clients) > 0 {
		client := s.Clients[0]
		client["password"] = strings.TrimSpace(stringValue(client["password"]))
		delete(client, "flow")
		delete(client, "id")
	}
	s.StreamSettings = rebuildTransportSettings(rawStream, transportOptions{
		allowTLS:     true,
		allowReality: true,
	})
}

func (s *InboundSchema) normalizeShadowsocks(rawSettings map[string]interface{}, rawClients []map[string]interface{}) {
	password := strings.TrimSpace(stringValue(rawSettings["password"]))
	if password == "" && len(rawClients) > 0 {
		password = strings.TrimSpace(stringValue(rawClients[0]["password"]))
	}
	method := strings.TrimSpace(strings.ToLower(stringValue(rawSettings["method"])))
	if method == "" {
		method = "aes-256-gcm"
	}
	network := strings.TrimSpace(strings.ToLower(stringValue(rawSettings["network"])))
	if network == "" {
		network = "tcp,udp"
	}
	s.Settings["password"] = password
	s.Settings["method"] = method
	s.Settings["network"] = network
	s.Clients = nil
	s.StreamSettings = map[string]interface{}{}
}

func (s *InboundSchema) normalizeSocks(rawSettings map[string]interface{}) {
	s.Clients = nil
	auth := strings.TrimSpace(strings.ToLower(stringValue(rawSettings["auth"])))
	switch auth {
	case "", "noauth":
		s.Settings["auth"] = "noauth"
		delete(s.Settings, "accounts")
	case "password":
		s.Settings["auth"] = "password"
		if accounts := filterAccounts(rawSettings["accounts"]); len(accounts) > 0 {
			s.Settings["accounts"] = accounts
		}
	default:
		s.Settings["auth"] = auth
	}
	if strings.TrimSpace(stringValue(rawSettings["ip"])) == "" {
		s.Settings["ip"] = "127.0.0.1"
	} else {
		s.Settings["ip"] = strings.TrimSpace(stringValue(rawSettings["ip"]))
	}
	s.Settings["udp"] = boolValueOrDefault(rawSettings["udp"], true)
	s.StreamSettings = map[string]interface{}{}
}

func (s *InboundSchema) normalizeHTTP(rawSettings map[string]interface{}, rawStream map[string]interface{}) {
	s.Clients = nil
	s.Settings["allowTransparent"] = boolValue(rawSettings["allowTransparent"])
	authEnabled := boolValue(rawSettings["auth"])
	if !authEnabled {
		delete(s.Settings, "accounts")
	} else if accounts := filterAccounts(rawSettings["accounts"]); len(accounts) > 0 {
		s.Settings["accounts"] = accounts
	}
	s.Settings["auth"] = authEnabled
	s.StreamSettings = rebuildHTTPStreamSettings(rawStream)
}

func (s *InboundSchema) normalizeDokodemo(rawSettings map[string]interface{}) {
	s.Clients = nil
	s.Settings["address"] = strings.TrimSpace(stringValue(rawSettings["address"]))
	s.Settings["port"] = numericValue(rawSettings["port"])
	network := strings.TrimSpace(strings.ToLower(stringValue(rawSettings["network"])))
	if network == "" {
		network = "tcp,udp"
	}
	s.Settings["network"] = network
	s.Settings["followRedirect"] = boolValue(rawSettings["followRedirect"])
	s.StreamSettings = map[string]interface{}{}
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

func cloneClients(input []map[string]interface{}) []map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(input))
	for _, item := range input {
		out = append(out, cloneMap(item))
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
