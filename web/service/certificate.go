package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
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
	LastRenew int64  `json:"lastRenewAt"`
	AutoRenew bool   `json:"autoRenew"`
	Type      string `json:"type"`
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
