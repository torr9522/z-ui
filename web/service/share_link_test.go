package service

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateVLESSRealityXHTTPShareLink(t *testing.T) {
	settings := map[string]interface{}{
		"clients": []interface{}{
			map[string]interface{}{
				"id":   "11111111-1111-1111-1111-111111111111",
				"flow": "xtls-rprx-vision",
			},
		},
		"decryption": "none",
	}
	stream := map[string]interface{}{
		"network":  "xhttp",
		"security": "reality",
		"realitySettings": map[string]interface{}{
			"publicKey":   "pubkey",
			"shortIds":    []interface{}{"abcd1234"},
			"serverNames": []interface{}{"example.com"},
			"spiderX":     "/hello",
		},
		"xhttpSettings": map[string]interface{}{
			"path": "/xhttp",
			"host": "example.com",
			"mode": "stream-one",
		},
	}

	link, err := generateShareLink("vless", 443, "45.77.246.87", "rc-vless", settings, stream)
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}

	u, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse link: %v", err)
	}
	if u.Scheme != "vless" {
		t.Fatalf("unexpected scheme: %s", u.Scheme)
	}
	query := u.Query()
	if query.Get("type") != "xhttp" ||
		query.Get("security") != "reality" ||
		query.Get("pbk") != "pubkey" ||
		query.Get("sid") != "abcd1234" ||
		query.Get("spx") != "/hello" ||
		query.Get("mode") != "stream-one" ||
		query.Get("flow") != "xtls-rprx-vision" {
		t.Fatalf("unexpected query: %v", query)
	}
}

func TestGenerateTrojanShareLinkOmitsFlow(t *testing.T) {
	settings := map[string]interface{}{
		"clients": []interface{}{
			map[string]interface{}{
				"password": "secret-pass",
				"flow":     "xtls-rprx-vision",
			},
		},
	}
	stream := map[string]interface{}{
		"network":  "ws",
		"security": "tls",
		"tlsSettings": map[string]interface{}{
			"serverName": "trojan.example.com",
		},
		"wsSettings": map[string]interface{}{
			"path": "/ws",
			"headers": map[string]interface{}{
				"Host": "trojan.example.com",
			},
		},
	}

	link, err := generateShareLink("trojan", 443, "45.77.246.87", "rc-trojan", settings, stream)
	if err != nil {
		t.Fatalf("generate share link: %v", err)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse link: %v", err)
	}
	query := u.Query()
	if query.Get("type") != "ws" ||
		query.Get("security") != "tls" ||
		query.Get("sni") != "trojan.example.com" ||
		query.Get("path") != "/ws" ||
		query.Get("host") != "trojan.example.com" {
		t.Fatalf("unexpected query: %v", query)
	}
	if query.Get("flow") != "" {
		t.Fatalf("trojan share link should omit flow, got %q", query.Get("flow"))
	}
}

func generateShareLink(protocol string, port int, address string, remark string, settings map[string]interface{}, stream map[string]interface{}) (string, error) {
	switch protocol {
	case "vless":
		client := firstMap(settings["clients"])
		u := &url.URL{
			Scheme:   "vless",
			User:     url.User(valueString(client["id"])),
			Host:     netJoin(address, port),
			Fragment: url.QueryEscape(remark),
		}
		query := url.Values{}
		query.Set("type", valueString(stream["network"]))
		query.Set("security", valueString(stream["security"]))
		if xhttp, ok := stream["xhttpSettings"].(map[string]interface{}); ok {
			query.Set("path", valueString(xhttp["path"]))
			query.Set("host", valueString(xhttp["host"]))
			query.Set("mode", valueString(xhttp["mode"]))
		}
		if reality, ok := stream["realitySettings"].(map[string]interface{}); ok {
			query.Set("pbk", valueString(reality["publicKey"]))
			query.Set("sid", firstString(reality["shortIds"]))
			query.Set("spx", valueString(reality["spiderX"]))
			query.Set("sni", firstString(reality["serverNames"]))
		}
		if flow := valueString(client["flow"]); flow != "" {
			query.Set("flow", flow)
		}
		u.RawQuery = query.Encode()
		return u.String(), nil
	case "trojan":
		client := firstMap(settings["clients"])
		password := valueString(client["password"])
		u := &url.URL{
			Scheme:   "trojan",
			User:     url.User(password),
			Host:     netJoin(address, port),
			Fragment: url.QueryEscape(remark),
		}
		query := url.Values{}
		query.Set("type", valueString(stream["network"]))
		query.Set("security", "tls")
		if tlsSettings, ok := stream["tlsSettings"].(map[string]interface{}); ok {
			serverName := valueString(tlsSettings["serverName"])
			if serverName != "" {
				u.Host = netJoin(serverName, port)
				query.Set("sni", serverName)
			}
		}
		if ws, ok := stream["wsSettings"].(map[string]interface{}); ok {
			query.Set("path", valueString(ws["path"]))
			if headers, ok := ws["headers"].(map[string]interface{}); ok {
				query.Set("host", valueString(headers["Host"]))
			}
		}
		u.RawQuery = query.Encode()
		return u.String(), nil
	default:
		return "", nil
	}
}

func netJoin(host string, port int) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(host), port)
}

func firstMap(value interface{}) map[string]interface{} {
	list, ok := value.([]interface{})
	if !ok || len(list) == 0 {
		return map[string]interface{}{}
	}
	item, _ := list[0].(map[string]interface{})
	return item
}

func firstString(value interface{}) string {
	list, ok := value.([]interface{})
	if !ok || len(list) == 0 {
		return ""
	}
	return valueString(list[0])
}

func valueString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}
