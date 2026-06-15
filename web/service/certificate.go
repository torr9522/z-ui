package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"x-ui/web/entity"
)

const certificatesDir = "/etc/x-ui/certs"

type certificateMeta struct {
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Issuer    string `json:"issuer"`
	CertFile  string `json:"certFile"`
	KeyFile   string `json:"keyFile"`
	CreatedAt int64  `json:"createdAt"`
	ExpireAt  int64  `json:"expireAt"`
	AutoRenew bool   `json:"autoRenew"`
}

type CertificateService struct{}

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
		dir := filepath.Join(certificatesDir, name)
		metaPath := filepath.Join(dir, "meta.json")
		certPath := filepath.Join(dir, "fullchain.pem")
		keyPath := filepath.Join(dir, "privkey.pem")
		if _, err := os.Stat(certPath); err != nil {
			continue
		}
		if _, err := os.Stat(keyPath); err != nil {
			continue
		}

		meta := &certificateMeta{
			Name:     name,
			Domain:   name,
			Issuer:   "manual",
			CertFile: certPath,
			KeyFile:  keyPath,
			ExpireAt: 0,
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

		certificates = append(certificates, &entity.Certificate{
			Name:     meta.Name,
			Domain:   meta.Domain,
			CertFile: meta.CertFile,
			KeyFile:  meta.KeyFile,
			ExpireAt: meta.ExpireAt,
			Issuer:   meta.Issuer,
		})
	}

	sort.Slice(certificates, func(i, j int) bool {
		return certificates[i].Name < certificates[j].Name
	})
	return certificates, nil
}
