package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"x-ui/database/model"
	"x-ui/util/json_util"
	"x-ui/xray"

	"github.com/xtls/xray-core/common/uuid"
)

var strictUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type commonModule struct {
	name            string
	supportsClients bool
}

func (m *commonModule) Name() string {
	return m.name
}

func (m *commonModule) Migrate(inbound *model.Inbound) (*model.Inbound, error) {
	return inbound, nil
}

func (m *commonModule) FormSchema() FormSchema {
	return FormSchema{
		Protocol:        m.name,
		SupportsClients: m.supportsClients,
	}
}

func normalizeInboundProtocol(inbound *model.Inbound, name string) {
	inbound.Protocol = model.Protocol(name)
}

func buildInboundConfig(inbound *model.Inbound, settings string, streamSettings string) *xray.InboundConfig {
	listen := inbound.Listen
	if listen != "" {
		listen = fmt.Sprintf("\"%v\"", listen)
	}
	return &xray.InboundConfig{
		Listen:         json_util.RawMessage(listen),
		Port:           inbound.Port,
		Protocol:       string(inbound.Protocol),
		Settings:       json_util.RawMessage(settings),
		StreamSettings: json_util.RawMessage(streamSettings),
		Tag:            inbound.Tag,
		Sniffing:       json_util.RawMessage(inbound.Sniffing),
	}
}

func decodeObject(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]interface{}{}, nil
	}
	out := make(map[string]interface{})
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func encodeObject(obj map[string]interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeStreamSettings(protocolName string, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return raw, nil
	}
	stream, err := decodeObject(raw)
	if err != nil {
		return "", err
	}

	network := strings.ToLower(stringValue(stream["network"]))
	security := strings.ToLower(stringValue(stream["security"]))

	if network == "splithttp" {
		network = "xhttp"
		stream["network"] = "xhttp"
	}
	if network == "http" {
		stream["network"] = "xhttp"
		xhttpSettings := map[string]interface{}{}
		if existing, ok := stream["xhttpSettings"].(map[string]interface{}); ok {
			xhttpSettings = existing
		} else if existing, ok := stream["splithttpSettings"].(map[string]interface{}); ok {
			xhttpSettings = existing
		}
		if httpSettings, ok := stream["httpSettings"].(map[string]interface{}); ok {
			if _, exists := xhttpSettings["path"]; !exists {
				xhttpSettings["path"] = stringValue(httpSettings["path"])
			}
			if _, exists := xhttpSettings["host"]; !exists {
				xhttpSettings["host"] = firstString(httpSettings["host"])
			}
			if _, exists := xhttpSettings["mode"]; !exists {
				xhttpSettings["mode"] = "stream-one"
			}
		}
		stream["xhttpSettings"] = xhttpSettings
		delete(stream, "httpSettings")
		network = "xhttp"
	}

	if security == "xtls" {
		if protocolName != "vless" && protocolName != "trojan" {
			return "", errors.New("legacy xtls is only supported for vless and trojan compatibility")
		}
		stream["security"] = "tls"
		if _, ok := stream["tlsSettings"]; !ok {
			if xtlsSettings, has := stream["xtlsSettings"]; has {
				stream["tlsSettings"] = xtlsSettings
			}
		}
		delete(stream, "xtlsSettings")
		security = "tls"
	}

	if xtlsSettings, ok := stream["xtlsSettings"]; ok {
		if _, exists := stream["tlsSettings"]; !exists {
			stream["tlsSettings"] = xtlsSettings
		}
		delete(stream, "xtlsSettings")
	}

	if network == "xhttp" {
		if _, ok := stream["xhttpSettings"]; !ok {
			if settings, has := stream["splithttpSettings"]; has {
				stream["xhttpSettings"] = settings
			} else {
				return "", errors.New("xhttp requires xhttpSettings or compatible legacy httpSettings")
			}
		}
		delete(stream, "splithttpSettings")
	}

	if security == "reality" {
		switch network {
		case "tcp", "xhttp", "grpc":
		default:
			return "", errors.New("reality only supports tcp, xhttp, and grpc")
		}
		if _, ok := stream["realitySettings"]; !ok {
			return "", errors.New("reality requires realitySettings")
		}
	}

	if security == "tls" || security == "reality" {
		if flowCarrier, ok := stream["tlsSettings"].(map[string]interface{}); ok {
			delete(flowCarrier, "flow")
		}
	}

	return encodeObject(stream)
}

func emptyObject() string {
	return "{}"
}

func clearTransportState(inbound *model.Inbound) {
	inbound.StreamSettings = emptyObject()
	inbound.Sniffing = emptyObject()
}

func normalizeVisionFlows(raw string) (string, error) {
	settings, err := decodeObject(raw)
	if err != nil {
		return "", err
	}

	normalizeClientsFlow(settings, "clients")
	return encodeObject(settings)
}

func normalizeVLESSSettings(raw string) (string, error) {
	settings, err := decodeObject(raw)
	if err != nil {
		return "", err
	}
	decryption, exists := settings["decryption"]
	if !exists || strings.TrimSpace(stringValue(decryption)) == "" || strings.EqualFold(stringValue(decryption), "null") {
		settings["decryption"] = "none"
	}
	normalizeClientsFlow(settings, "clients")
	return encodeObject(settings)
}

func normalizeVMessSettings(raw string) (string, error) {
	settings, err := decodeObject(raw)
	if err != nil {
		return "", err
	}
	clients, ok := settings["clients"].([]interface{})
	if !ok {
		return encodeObject(settings)
	}
	for _, item := range clients {
		client, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		client["alterId"] = 0
	}
	return encodeObject(settings)
}

func normalizeTrojanFlows(raw string) (string, error) {
	settings, err := decodeObject(raw)
	if err != nil {
		return "", err
	}

	removeClientsFlow(settings, "clients")
	return encodeObject(settings)
}

func normalizeClientsFlow(settings map[string]interface{}, field string) {
	clients, ok := settings[field].([]interface{})
	if !ok {
		return
	}
	for _, item := range clients {
		client, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		flow := strings.ToLower(stringValue(client["flow"]))
		switch flow {
		case "", "xtls-rprx-vision":
			continue
		case "xtls-rprx-origin", "xtls-rprx-direct":
			client["flow"] = "xtls-rprx-vision"
		case "xtls-rprx-splice":
			delete(client, "flow")
		default:
			if strings.HasPrefix(flow, "xtls-rprx-") {
				delete(client, "flow")
			}
		}
	}
}

func removeClientsFlow(settings map[string]interface{}, field string) {
	clients, ok := settings[field].([]interface{})
	if !ok {
		return
	}
	for _, item := range clients {
		client, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		delete(client, "flow")
	}
}

func validateClients(settings map[string]interface{}, field string) error {
	clients, ok := settings[field].([]interface{})
	if !ok || len(clients) == 0 {
		return fmt.Errorf("%s must contain at least one client", field)
	}
	return nil
}

func validateClientUUIDs(settings map[string]interface{}, field string) error {
	clients, ok := settings[field].([]interface{})
	if !ok || len(clients) == 0 {
		return fmt.Errorf("%s must contain at least one client", field)
	}
	for index, item := range clients {
		client, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", field, index)
		}
		id := strings.TrimSpace(stringValue(client["id"]))
		if id == "" {
			return fmt.Errorf("%s[%d].id must not be empty", field, index)
		}
		if !strictUUIDPattern.MatchString(id) {
			return fmt.Errorf("%s[%d].id must be a valid UUID", field, index)
		}
		if _, err := uuid.ParseString(id); err != nil {
			return fmt.Errorf("%s[%d].id must be a valid UUID", field, index)
		}
	}
	return nil
}

func validateClientPasswords(settings map[string]interface{}, field string) error {
	clients, ok := settings[field].([]interface{})
	if !ok || len(clients) == 0 {
		return fmt.Errorf("%s must contain at least one client", field)
	}
	for index, item := range clients {
		client, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", field, index)
		}
		if strings.TrimSpace(stringValue(client["password"])) == "" {
			return fmt.Errorf("%s[%d].password must not be empty", field, index)
		}
	}
	return nil
}

func validateRealityRequired(streamSettings string) error {
	stream, err := decodeObject(streamSettings)
	if err != nil {
		return err
	}
	if strings.ToLower(stringValue(stream["security"])) != "reality" {
		return nil
	}
	reality, ok := stream["realitySettings"].(map[string]interface{})
	if !ok {
		return errors.New("reality requires realitySettings")
	}
	if strings.TrimSpace(stringValue(reality["privateKey"])) == "" {
		return errors.New("reality privateKey must not be empty")
	}
	if strings.TrimSpace(stringValue(reality["dest"])) == "" {
		return errors.New("reality dest must not be empty")
	}
	return nil
}

func firstString(value interface{}) string {
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if str := stringValue(item); str != "" {
				return str
			}
		}
	case []string:
		for _, item := range typed {
			if item != "" {
				return item
			}
		}
	default:
		return stringValue(value)
	}
	return ""
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return strings.Trim(string(data), "\"")
	}
}

func boolValue(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func interfaceSlice(value interface{}) []interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	return items
}
