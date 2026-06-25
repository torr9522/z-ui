package entity

type Certificate struct {
	Name          string `json:"name"`
	Domain        string `json:"domain"`
	CertFile      string `json:"certFile"`
	KeyFile       string `json:"keyFile"`
	ExpireAt      int64  `json:"expireAt"`
	Issuer        string `json:"issuer"`
	Type          string `json:"type"`
	AutoRenew     bool   `json:"autoRenew"`
	DaysRemaining int64  `json:"daysRemaining"`
}

type DiscoveredCertificate struct {
	Domain          string `json:"domain"`
	CertPath        string `json:"certPath"`
	KeyPath         string `json:"keyPath"`
	Source          string `json:"source"`
	Issuer          string `json:"issuer"`
	NotBefore       string `json:"notBefore"`
	NotAfter        string `json:"notAfter"`
	Expired         bool   `json:"expired"`
	AlreadyImported bool   `json:"alreadyImported"`
}
