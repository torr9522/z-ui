package web

import (
	"net/http"
	"testing"
)

func TestBuildSessionOptions(t *testing.T) {
	options := buildSessionOptions("/xui/", true)
	if options.Path != "/xui/" {
		t.Fatalf("unexpected path: %q", options.Path)
	}
	if options.MaxAge != 12*60*60 {
		t.Fatalf("unexpected max age: %d", options.MaxAge)
	}
	if !options.HttpOnly {
		t.Fatal("expected HttpOnly")
	}
	if !options.Secure {
		t.Fatal("expected Secure")
	}
	if options.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected same site: %v", options.SameSite)
	}

	options = buildSessionOptions("/", false)
	if options.Secure {
		t.Fatal("expected insecure cookie for http mode")
	}
}
