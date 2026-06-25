package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"x-ui/web/entity"
)

const certificatesDir = "/etc/x-ui/certs"

var discoverRoots = []discoverRoot{
	{base: "/etc/letsencrypt/live", source: "letsencrypt"},
	{base: "/root/.acme.sh", source: "acme.sh"},
}

type discoverRoot struct {
	base   string
	source string
}

type certificateMeta struct {
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Issuer    string `json:"issuer"`
	CertFile  string `json:"certFile"`
	KeyFile   string `json:"keyFile"`
	CreatedAt int64  `json:"createdAt"`
	ExpireAt  int64  `json:"expireAt"`
	LastRenew int64  `json:"lastRenewAt"`
	AutoRenew bool   `json:"autoRenew"`
	Type      string `json:"type"`
}

type ImportDiscoveredCertificateRequest struct {
	CertPath string `form:"certPath" json:"certPath"`
	KeyPath  string `form:"keyPath"  json:"keyPath"`
	Name     string `form:"name"     json:"name"`
}

type CertificateService struct{}

func isIgnoredCertificateDir(name string) bool {
	if strings.HasPrefix(name, "deleted.") ||
		strings.HasPrefix(name, "backup.") ||
		strings.HasPrefix(name, "tmp.") ||
		strings.HasPrefix(name, "test.") {
		return true
	}
	return strings.Contains(name, ".bak.")
}

func sanitizeCertificateName(input string) string {
	var b strings.Builder
	input = strings.TrimSpace(input)
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	name := strings.Trim(b.String(), "._-")
	name = strings.ReplaceAll(name, "..", ".")
	return name
}

func certDir(name string) string {
	return filepath.Join(certificatesDir, name)
}

func certMetaPath(name string) string {
	return filepath.Join(certDir(name), "meta.json")
}

func isSubPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func resolvePath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func isAllowedDiscoverPath(path string) bool {
	path = filepath.Clean(path)
	allowedBases := []string{
		"/etc/letsencrypt/live",
		"/etc/letsencrypt/archive",
		"/root/.acme.sh",
	}
	for _, homeRoot := range userAcmeHomeRoots() {
		allowedBases = append(allowedBases, homeRoot)
	}
	for _, base := range allowedBases {
		if isSubPath(base, path) {
			return true
		}
	}
	return false
}

func isAllowedOriginalAndResolvedPath(original string, resolved string) bool {
	original = filepath.Clean(original)
	resolved = filepath.Clean(resolved)
	if isAllowedDiscoverPath(original) && isAllowedDiscoverPath(resolved) {
		return true
	}
	return isSubPath("/etc/letsencrypt/live", original) && isSubPath("/etc/letsencrypt/archive", resolved)
}

func userAcmeHomeRoots() []string {
	entries, err := os.ReadDir("/home")
	if err != nil {
		return nil
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roots = append(roots, filepath.Join("/home", entry.Name(), ".acme.sh"))
	}
	return roots
}

func issuerName(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	if cert.Issuer.CommonName != "" {
		return cert.Issuer.CommonName
	}
	return cert.Issuer.String()
}

func readCertificatePair(certPath, keyPath string) (*x509.Certificate, []byte, []byte, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, nil, err
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return nil, nil, nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, nil, errors.New("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, nil, err
	}
	return cert, certPEM, keyPEM, nil
}

func normalizeAcmeKeyPath(dir string) string {
	base := filepath.Base(dir)
	switch {
	case strings.HasSuffix(base, "_ecc"):
		domain := strings.TrimSuffix(base, "_ecc")
		return filepath.Join(dir, domain+".key")
	default:
		return filepath.Join(dir, base+".key")
	}
}

func importMetadata(name string, certFile string, keyFile string, cert *x509.Certificate, source string) *certificateMeta {
	issuer := source
	if value := issuerName(cert); value != "" {
		issuer = value
	}
	return &certificateMeta{
		Name:      name,
		Domain:    cert.Subject.CommonName,
		Issuer:    issuer,
		CertFile:  certFile,
		KeyFile:   keyFile,
		CreatedAt: time.Now().Unix(),
		ExpireAt:  cert.NotAfter.Unix(),
		LastRenew: 0,
		AutoRenew: false,
		Type:      "imported",
	}
}

func writeCertificateMeta(path string, meta *certificateMeta) error {
	payload, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, payload, 0600); err != nil {
		return err
	}
	return nil
}

func loadCertificateMeta(name string, certPath string, keyPath string) *certificateMeta {
	metaPath := certMetaPath(name)
	meta := &certificateMeta{
		Name:     name,
		Domain:   name,
		Issuer:   "manual",
		CertFile: certPath,
		KeyFile:  keyPath,
		ExpireAt: 0,
		Type:     "imported",
	}
	if data, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(data, meta)
	}
	if meta.Name == "" {
		meta.Name = name
	}
	if meta.Domain == "" {
		meta.Domain = name
	}
	if meta.CertFile == "" {
		meta.CertFile = certPath
	}
	if meta.KeyFile == "" {
		meta.KeyFile = keyPath
	}
	if meta.Issuer == "" {
		meta.Issuer = "manual"
	}
	if meta.Type == "" {
		meta.Type = "imported"
	}
	return meta
}

func (s *CertificateService) List() ([]*entity.Certificate, error) {
	entries, err := os.ReadDir(certificatesDir)
	if os.IsNotExist(err) {
		return []*entity.Certificate{}, nil
	}
	if err != nil {
		return nil, err
	}

	certificates := make([]*entity.Certificate, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isIgnoredCertificateDir(name) {
			continue
		}
		dir := filepath.Join(certificatesDir, name)
		certPath := filepath.Join(dir, "fullchain.pem")
		keyPath := filepath.Join(dir, "privkey.pem")
		if _, err := os.Stat(certPath); err != nil {
			continue
		}
		if _, err := os.Stat(keyPath); err != nil {
			continue
		}

		meta := loadCertificateMeta(name, certPath, keyPath)
		daysRemaining := int64(0)
		if meta.ExpireAt > 0 {
			daysRemaining = (meta.ExpireAt - time.Now().Unix()) / 86400
			if daysRemaining < 0 {
				daysRemaining = 0
			}
		}

		certificates = append(certificates, &entity.Certificate{
			Name:          meta.Name,
			Domain:        meta.Domain,
			CertFile:      meta.CertFile,
			KeyFile:       meta.KeyFile,
			ExpireAt:      meta.ExpireAt,
			Issuer:        meta.Issuer,
			Type:          meta.Type,
			AutoRenew:     meta.AutoRenew,
			DaysRemaining: daysRemaining,
		})
	}

	sort.Slice(certificates, func(i, j int) bool {
		return certificates[i].Name < certificates[j].Name
	})
	return certificates, nil
}

func (s *CertificateService) Discover() ([]*entity.DiscoveredCertificate, error) {
	imported, err := s.List()
	if err != nil {
		return nil, err
	}
	importedPaths := make(map[string]struct{}, len(imported)*2)
	for _, item := range imported {
		importedPaths[filepath.Clean(item.CertFile)] = struct{}{}
		importedPaths[filepath.Clean(item.KeyFile)] = struct{}{}
	}

	roots := append([]discoverRoot{}, discoverRoots...)
	for _, base := range userAcmeHomeRoots() {
		roots = append(roots, discoverRoot{base: base, source: "acme.sh"})
	}

	seen := map[string]struct{}{}
	result := make([]*entity.DiscoveredCertificate, 0)
	for _, root := range roots {
		items, err := discoverFromRoot(root.base, root.source, importedPaths, seen)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].AlreadyImported != result[j].AlreadyImported {
			return !result[i].AlreadyImported
		}
		if result[i].Domain != result[j].Domain {
			return result[i].Domain < result[j].Domain
		}
		return result[i].CertPath < result[j].CertPath
	})
	return result, nil
}

func discoverFromRoot(base string, source string, importedPaths map[string]struct{}, seen map[string]struct{}) ([]*entity.DiscoveredCertificate, error) {
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return []*entity.DiscoveredCertificate{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]*entity.DiscoveredCertificate, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(base, entry.Name())
		var certPath, keyPath string
		switch source {
		case "letsencrypt":
			certPath = filepath.Join(dir, "fullchain.pem")
			keyPath = filepath.Join(dir, "privkey.pem")
		default:
			certPath = filepath.Join(dir, "fullchain.cer")
			keyPath = normalizeAcmeKeyPath(dir)
		}
		if _, err := os.Stat(certPath); err != nil {
			continue
		}
		if _, err := os.Stat(keyPath); err != nil {
			continue
		}
		resolvedCert, err := resolvePath(certPath)
		if err != nil || !isAllowedOriginalAndResolvedPath(certPath, resolvedCert) {
			continue
		}
		resolvedKey, err := resolvePath(keyPath)
		if err != nil || !isAllowedOriginalAndResolvedPath(keyPath, resolvedKey) {
			continue
		}
		key := resolvedCert + "\x00" + resolvedKey
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		cert, _, _, err := readCertificatePair(resolvedCert, resolvedKey)
		if err != nil {
			continue
		}
		domain := strings.TrimSpace(cert.Subject.CommonName)
		if domain == "" {
			domain = entry.Name()
		}
		_, importedCert := importedPaths[resolvedCert]
		_, importedKey := importedPaths[resolvedKey]
		result = append(result, &entity.DiscoveredCertificate{
			Domain:          domain,
			CertPath:        filepath.Clean(certPath),
			KeyPath:         filepath.Clean(keyPath),
			Source:          source,
			Issuer:          issuerName(cert),
			NotBefore:       cert.NotBefore.Format(time.RFC3339),
			NotAfter:        cert.NotAfter.Format(time.RFC3339),
			Expired:         time.Now().After(cert.NotAfter),
			AlreadyImported: importedCert || importedKey,
		})
	}
	return result, nil
}

func (s *CertificateService) ImportDiscovered(req *ImportDiscoveredCertificateRequest) (*entity.Certificate, error) {
	if req == nil {
		return nil, errors.New("missing import request")
	}
	certPath := filepath.Clean(strings.TrimSpace(req.CertPath))
	keyPath := filepath.Clean(strings.TrimSpace(req.KeyPath))
	resolvedCert, err := resolvePath(certPath)
	if err != nil {
		return nil, errors.New("certificate file not found")
	}
	resolvedKey, err := resolvePath(keyPath)
	if err != nil {
		return nil, errors.New("private key file not found")
	}
	if !isAllowedOriginalAndResolvedPath(certPath, resolvedCert) || !isAllowedOriginalAndResolvedPath(keyPath, resolvedKey) {
		return nil, errors.New("certificate path is outside allowed discovery roots")
	}

	cert, certPEM, keyPEM, err := readCertificatePair(resolvedCert, resolvedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate pair: %w", err)
	}

	name := sanitizeCertificateName(req.Name)
	if name == "" {
		name = sanitizeCertificateName(cert.Subject.CommonName)
	}
	if name == "" {
		return nil, errors.New("certificate name is invalid")
	}

	targetDir := certDir(name)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return nil, err
	}
	certFile := filepath.Join(targetDir, "fullchain.pem")
	keyFile := filepath.Join(targetDir, "privkey.pem")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return nil, err
	}
	meta := importMetadata(name, certFile, keyFile, cert, detectImportSource(certPath))
	if meta.Domain == "" {
		meta.Domain = name
	}
	if err := writeCertificateMeta(certMetaPath(name), meta); err != nil {
		return nil, err
	}
	return &entity.Certificate{
		Name:          meta.Name,
		Domain:        meta.Domain,
		CertFile:      meta.CertFile,
		KeyFile:       meta.KeyFile,
		ExpireAt:      meta.ExpireAt,
		Issuer:        meta.Issuer,
		Type:          meta.Type,
		AutoRenew:     meta.AutoRenew,
		DaysRemaining: maxDaysRemaining(meta.ExpireAt),
	}, nil
}

func detectImportSource(certPath string) string {
	switch {
	case isSubPath("/etc/letsencrypt/live", certPath):
		return "letsencrypt"
	case isSubPath("/root/.acme.sh", certPath):
		return "acme.sh"
	default:
		for _, base := range userAcmeHomeRoots() {
			if isSubPath(base, certPath) {
				return "acme.sh"
			}
		}
	}
	return "manual"
}

func maxDaysRemaining(expireAt int64) int64 {
	if expireAt <= 0 {
		return 0
	}
	days := (expireAt - time.Now().Unix()) / 86400
	if days < 0 {
		return 0
	}
	return days
}
