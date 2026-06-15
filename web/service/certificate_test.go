package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCertificateServiceList(t *testing.T) {
	root := filepath.Join("/etc/x-ui", "certs", "service-test")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatalf("mkdir cert root: %v", err)
	}
	defer os.RemoveAll(root)

	if err := os.WriteFile(filepath.Join(root, "fullchain.pem"), []byte("test-cert"), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "privkey.pem"), []byte("test-key"), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	meta := `{
  "name": "service-test",
  "domain": "service.example.com",
  "issuer": "manual",
  "certFile": "/etc/x-ui/certs/service-test/fullchain.pem",
  "keyFile": "/etc/x-ui/certs/service-test/privkey.pem",
  "type": "domain",
  "createdAt": 0,
  "expireAt": 4102444800,
  "autoRenew": true
}`
	if err := os.WriteFile(filepath.Join(root, "meta.json"), []byte(meta), 0600); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	service := CertificateService{}
	certificates, err := service.List()
	if err != nil {
		t.Fatalf("list certificates: %v", err)
	}

	var found bool
	for _, item := range certificates {
		if item.Name != "service-test" {
			continue
		}
		found = true
		if item.Domain != "service.example.com" {
			t.Fatalf("unexpected domain: %s", item.Domain)
		}
		if item.Issuer != "manual" {
			t.Fatalf("unexpected issuer: %s", item.Issuer)
		}
		if item.CertFile != "/etc/x-ui/certs/service-test/fullchain.pem" {
			t.Fatalf("unexpected cert file: %s", item.CertFile)
		}
		if item.KeyFile != "/etc/x-ui/certs/service-test/privkey.pem" {
			t.Fatalf("unexpected key file: %s", item.KeyFile)
		}
		if item.ExpireAt != 4102444800 {
			t.Fatalf("unexpected expireAt: %d", item.ExpireAt)
		}
		if item.Type != "domain" {
			t.Fatalf("unexpected type: %s", item.Type)
		}
		if !item.AutoRenew {
			t.Fatal("expected autoRenew to be true")
		}
		if item.DaysRemaining <= 0 {
			t.Fatalf("expected positive days remaining, got %d", item.DaysRemaining)
		}
	}
	if !found {
		t.Fatal("expected service-test certificate in list")
	}
}
