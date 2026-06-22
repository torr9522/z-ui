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

type transportOptions struct {
	allowTLS     bool
	allowReality bool
}

func rebuildTransportSettings(raw map[string]interface{}, opts transportOptions) map[string]interface{} {
	out := map[string]interface{}{}
	if len(raw) == 0 {
		return out
	}

	network := normalizedNetwork(raw)
	if network != "" {
		out["network"] = network
	}

	security := normalizedSecurity(raw)
	switch security {
	case "tls":
		if opts.allowTLS {
			out["security"] = "tls"
			if tlsSettings := sanitizeTLSSettings(raw["tlsSettings"], raw["xtlsSettings"]); len(tlsSettings) > 0 {
				out["tlsSettings"] = tlsSettings
			}
		}
	case "reality":
		if opts.allowReality {
			out["security"] = "reality"
			if realitySettings := sanitizeRealitySettings(raw["realitySettings"]); len(realitySettings) > 0 {
				out["realitySettings"] = realitySettings
			}
		}
	}

	switch network {
	case "tcp":
		if tcpSettings := sanitizeTCPSettings(raw["tcpSettings"]); len(tcpSettings) > 0 {
			out["tcpSettings"] = tcpSettings
		}
	case "ws":
		if wsSettings := sanitizeWSSettings(raw["wsSettings"]); len(wsSettings) > 0 {
			out["wsSettings"] = wsSettings
		}
	case "xhttp":
		if xhttpSettings := sanitizeXHTTPSettings(raw); len(xhttpSettings) > 0 {
			out["xhttpSettings"] = xhttpSettings
		}
	case "grpc":
		if grpcSettings := sanitizeGRPCSettings(raw["grpcSettings"]); len(grpcSettings) > 0 {
			out["grpcSettings"] = grpcSettings
		}
	}

	return out
}

func rebuildHTTPStreamSettings(raw map[string]interface{}) map[string]interface{} {
	security := normalizedSecurity(raw)
	if security != "tls" {
		return map[string]interface{}{}
	}
	tlsSettings := sanitizeTLSSettings(raw["tlsSettings"], raw["xtlsSettings"])
	if len(tlsSettings) == 0 {
		return map[string]interface{}{"security": "tls"}
	}
	return map[string]interface{}{
		"security":    "tls",
		"tlsSettings": tlsSettings,
	}
}

func normalizedNetwork(raw map[string]interface{}) string {
	network := strings.ToLower(strings.TrimSpace(stringValue(raw["network"])))
	switch network {
	case "http", "splithttp":
		return "xhttp"
	case "tcp", "ws", "xhttp", "grpc":
		return network
	default:
		return "tcp"
	}
}

func normalizedSecurity(raw map[string]interface{}) string {
	security := strings.ToLower(strings.TrimSpace(stringValue(raw["security"])))
	if security == "xtls" {
		return "tls"
	}
	switch security {
	case "tls", "reality":
		return security
	default:
		return "none"
	}
}

func sanitizeTLSSettings(primary interface{}, legacy interface{}) map[string]interface{} {
	settings, ok := primary.(map[string]interface{})
	if !ok {
		settings, _ = legacy.(map[string]interface{})
	}
	if len(settings) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if serverName := strings.TrimSpace(stringValue(settings["serverName"])); serverName != "" {
		out["serverName"] = serverName
	}
	if certificates := sanitizeCertificateList(settings["certificates"]); len(certificates) > 0 {
		out["certificates"] = certificates
	}
	return out
}

func sanitizeRealitySettings(value interface{}) map[string]interface{} {
	settings, ok := value.(map[string]interface{})
	if !ok || len(settings) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{
		"show":        boolValue(settings["show"]),
		"privateKey":  strings.TrimSpace(stringValue(settings["privateKey"])),
		"publicKey":   strings.TrimSpace(stringValue(settings["publicKey"])),
		"spiderX":     strings.TrimSpace(stringValue(settings["spiderX"])),
		"dest":        strings.TrimSpace(stringValue(settings["dest"])),
		"fingerprint": strings.TrimSpace(stringValue(settings["fingerprint"])),
	}
	if out["spiderX"] == "" {
		out["spiderX"] = "/"
	}
	if out["fingerprint"] == "" {
		out["fingerprint"] = "chrome"
	}
	if shortIds := sanitizeStringList(settings["shortIds"]); len(shortIds) > 0 {
		out["shortIds"] = shortIds
	}
	if serverNames := sanitizeStringList(settings["serverNames"]); len(serverNames) > 0 {
		out["serverNames"] = serverNames
	}
	return out
}

func sanitizeTCPSettings(value interface{}) map[string]interface{} {
	settings, ok := value.(map[string]interface{})
	if !ok || len(settings) == 0 {
		return map[string]interface{}{}
	}
	header, _ := settings["header"].(map[string]interface{})
	headerType := strings.TrimSpace(strings.ToLower(stringValue(header["type"])))
	if headerType == "" {
		headerType = "none"
	}
	out := map[string]interface{}{
		"header": map[string]interface{}{
			"type": headerType,
		},
	}
	if headerType == "http" {
		request := map[string]interface{}{}
		if req, ok := header["request"].(map[string]interface{}); ok {
			request["version"] = firstNonEmptyString(req["version"], "1.1")
			request["method"] = firstNonEmptyString(req["method"], "GET")
			if paths := sanitizeStringList(req["path"]); len(paths) > 0 {
				request["path"] = paths
			} else {
				request["path"] = []string{"/"}
			}
			if headers := sanitizeHeaderMap(req["headers"]); len(headers) > 0 {
				request["headers"] = headers
			}
		}
		response := map[string]interface{}{}
		if resp, ok := header["response"].(map[string]interface{}); ok {
			response["version"] = firstNonEmptyString(resp["version"], "1.1")
			response["status"] = firstNonEmptyString(resp["status"], "200")
			response["reason"] = firstNonEmptyString(resp["reason"], "OK")
			if headers := sanitizeHeaderMap(resp["headers"]); len(headers) > 0 {
				response["headers"] = headers
			}
		}
		out["header"].(map[string]interface{})["request"] = request
		out["header"].(map[string]interface{})["response"] = response
	}
	return out
}

func sanitizeWSSettings(value interface{}) map[string]interface{} {
	settings, ok := value.(map[string]interface{})
	if !ok || len(settings) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{
		"path": firstNonEmptyString(settings["path"], "/"),
	}
	if headers := sanitizeHeaderMap(settings["headers"]); len(headers) > 0 {
		out["headers"] = headers
	}
	return out
}

func sanitizeXHTTPSettings(raw map[string]interface{}) map[string]interface{} {
	var settings map[string]interface{}
	switch {
	case raw["xhttpSettings"] != nil:
		settings, _ = raw["xhttpSettings"].(map[string]interface{})
	case raw["splithttpSettings"] != nil:
		settings, _ = raw["splithttpSettings"].(map[string]interface{})
	case raw["httpSettings"] != nil:
		settings, _ = raw["httpSettings"].(map[string]interface{})
	}
	if len(settings) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{
		"path": firstNonEmptyString(settings["path"], "/"),
		"host": strings.TrimSpace(firstString(settings["host"])),
		"mode": firstNonEmptyString(settings["mode"], "stream-one"),
	}
	if extra := strings.TrimSpace(stringValue(settings["extra"])); extra != "" {
		out["extra"] = extra
	}
	if headers := sanitizeHeaderMap(settings["headers"]); len(headers) > 0 {
		out["headers"] = headers
	}
	return out
}

func sanitizeGRPCSettings(value interface{}) map[string]interface{} {
	settings, ok := value.(map[string]interface{})
	if !ok || len(settings) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if serviceName := strings.TrimSpace(stringValue(settings["serviceName"])); serviceName != "" {
		out["serviceName"] = serviceName
	}
	return out
}

func sanitizeCertificateList(value interface{}) []interface{} {
	items := interfaceSlice(value)
	if len(items) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		certFile := strings.TrimSpace(stringValue(entry["certificateFile"]))
		keyFile := strings.TrimSpace(stringValue(entry["keyFile"]))
		if certFile != "" || keyFile != "" {
			out = append(out, map[string]interface{}{
				"certificateFile": certFile,
				"keyFile":         keyFile,
			})
			continue
		}
		cert := sanitizeStringList(entry["certificate"])
		key := sanitizeStringList(entry["key"])
		if len(cert) > 0 && len(key) > 0 {
			out = append(out, map[string]interface{}{
				"certificate": cert,
				"key":         key,
			})
		}
	}
	return out
}

func sanitizeStringList(value interface{}) []interface{} {
	items := interfaceSlice(value)
	if len(items) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(stringValue(item)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func sanitizeHeaderMap(value interface{}) map[string]interface{} {
	headers, ok := value.(map[string]interface{})
	if !ok || len(headers) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(headers))
	for key, item := range headers {
		switch typed := item.(type) {
		case []interface{}:
			values := sanitizeStringList(typed)
			if len(values) > 0 {
				out[key] = values
			}
		default:
			if trimmed := strings.TrimSpace(stringValue(typed)); trimmed != "" {
				out[key] = trimmed
			}
		}
	}
	return out
}

func filterAccounts(value interface{}) []interface{} {
	items := interfaceSlice(value)
	if len(items) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		account, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		user := strings.TrimSpace(stringValue(account["user"]))
		pass := strings.TrimSpace(stringValue(account["pass"]))
		if user == "" || pass == "" {
			continue
		}
		out = append(out, map[string]interface{}{
			"user": user,
			"pass": pass,
		})
	}
	return out
}

func numericValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		v, _ := typed.Int64()
		return int(v)
	case string:
		var out int
		_, _ = fmt.Sscanf(strings.TrimSpace(typed), "%d", &out)
		return out
	default:
		return 0
	}
}

func boolValueOrDefault(value interface{}, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return boolValue(value)
}

func firstNonEmptyString(value interface{}, fallback string) string {
	if trimmed := strings.TrimSpace(firstString(value)); trimmed != "" {
		return trimmed
	}
	return fallback
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
	if value == nil {
		return ""
	}
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
