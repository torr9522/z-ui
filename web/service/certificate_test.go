package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeTextFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func createSelfSignedPair(t *testing.T, dir string, domain string, certName string, keyName string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir pair dir: %v", err)
	}
	keyPath := filepath.Join(dir, keyName)
	certPath := filepath.Join(dir, certName)
	cmd := exec.Command(
		"openssl", "req", "-x509", "-nodes", "-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-subj", "/CN="+domain,
		"-days", "365",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("openssl req failed: %v\n%s", err, string(output))
	}
	return certPath, keyPath
}

func cleanupPaths(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("cleanup %s: %v", path, err)
		}
	}
}

func TestCertificateServiceList(t *testing.T) {
	root := filepath.Join("/etc/x-ui", "certs", "service-test")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatalf("mkdir cert root: %v", err)
	}
	defer cleanupPaths(t, root)

	certPath, keyPath := createSelfSignedPair(t, root, "service.example.com", "fullchain.pem", "privkey.pem")
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
	writeTextFile(t, filepath.Join(root, "meta.json"), meta, 0600)

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
		if item.CertFile != certPath {
			t.Fatalf("unexpected cert file: %s", item.CertFile)
		}
		if item.KeyFile != keyPath {
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

func TestIgnoredCertificateDirs(t *testing.T) {
	cases := map[string]bool{
		"deleted.example":       true,
		"example.bak.20260616":  true,
		"backup.example":        true,
		"tmp.example":           true,
		"test.example":          true,
		"example.com":           false,
		"imported.example.com":  false,
		"example.backup.domain": false,
	}

	for name, expected := range cases {
		if got := isIgnoredCertificateDir(name); got != expected {
			t.Fatalf("isIgnoredCertificateDir(%q) = %v, expected %v", name, got, expected)
		}
	}
}

func TestCertificateServiceDiscoverLetsEncrypt(t *testing.T) {
	leDir := "/etc/letsencrypt/live/discover-test.example.com"
	defer cleanupPaths(t, leDir)
	certPath, keyPath := createSelfSignedPair(t, leDir, "discover-test.example.com", "fullchain.pem", "privkey.pem")

	service := CertificateService{}
	items, err := service.Discover()
	if err != nil {
		t.Fatalf("discover certificates: %v", err)
	}

	var found bool
	for _, item := range items {
		if item.CertPath != certPath {
			continue
		}
		found = true
		if item.KeyPath != keyPath {
			t.Fatalf("unexpected key path: %s", item.KeyPath)
		}
		if item.Domain != "discover-test.example.com" {
			t.Fatalf("unexpected domain: %s", item.Domain)
		}
		if item.Source != "letsencrypt" {
			t.Fatalf("unexpected source: %s", item.Source)
		}
		if item.AlreadyImported {
			t.Fatal("expected discovered certificate to not be imported yet")
		}
		if item.NotBefore == "" || item.NotAfter == "" {
			t.Fatalf("expected certificate time bounds, got %#v", item)
		}
	}
	if !found {
		t.Fatal("expected letsencrypt certificate in discover results")
	}
}

func TestCertificateServiceImportDiscovered(t *testing.T) {
	acmeDir := "/root/.acme.sh/import-test.example.com"
	targetDir := filepath.Join("/etc/x-ui/certs", "import-test.example.com")
	defer cleanupPaths(t, acmeDir, targetDir)
	certPath, keyPath := createSelfSignedPair(t, acmeDir, "import-test.example.com", "fullchain.cer", "import-test.example.com.key")

	service := CertificateService{}
	certificate, err := service.ImportDiscovered(&ImportDiscoveredCertificateRequest{
		CertPath: certPath,
		KeyPath:  keyPath,
		Name:     "import-test.example.com",
	})
	if err != nil {
		t.Fatalf("import discovered certificate: %v", err)
	}

	if certificate.Name != "import-test.example.com" {
		t.Fatalf("unexpected imported name: %s", certificate.Name)
	}
	if certificate.CertFile != filepath.Join(targetDir, "fullchain.pem") {
		t.Fatalf("unexpected cert file: %s", certificate.CertFile)
	}
	if certificate.KeyFile != filepath.Join(targetDir, "privkey.pem") {
		t.Fatalf("unexpected key file: %s", certificate.KeyFile)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "meta.json")); err != nil {
		t.Fatalf("expected imported meta.json: %v", err)
	}

	items, err := service.List()
	if err != nil {
		t.Fatalf("list certificates after import: %v", err)
	}
	var found bool
	for _, item := range items {
		if item.Name == "import-test.example.com" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected imported certificate to appear in managed list")
	}
}
