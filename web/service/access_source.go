package service

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"x-ui/xray"

	"github.com/oschwald/maxminddb-golang"
)

const (
	accessSourceMaxReadBytes = 64 * 1024 * 1024
	accessSourceTimeLayout   = "2006/01/02 15:04:05.000000"
	accessSourceBatchEvery   = 30 * time.Second
	ipProfileTTL             = 24 * time.Hour
)

var accessSourceEventPattern = regexp.MustCompile(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d+)\s+from\s+(\S+)\s+accepted\s+.*?\[inbound-(\d+)\]`)

type AccessSourceService struct {
	settingService SettingService
}

type AccessSourceReport struct {
	Ports   []*PortUsageStat `json:"ports"`
	Message string           `json:"message"`
}

type PortUsageStat struct {
	Port      int            `json:"port"`
	UniqueIPs int            `json:"unique_ips"`
	TotalHits int64          `json:"total_hits"`
	LastSeen  int64          `json:"last_seen,omitempty"`
	IPs       []*IPUsageStat `json:"ips,omitempty"`
}

type IPUsageStat struct {
	IPProfile
	HitCount int64 `json:"hit_count,omitempty"`
	LastSeen int64 `json:"last_seen,omitempty"`
}

type IPProfile struct {
	IP          string `json:"ip"`
	Geo         string `json:"geo"`
	CountryCode string `json:"country_code,omitempty"`
	City        string `json:"city,omitempty"`
	ASN         uint   `json:"asn,omitempty"`
	ISP         string `json:"isp"`
	DeviceType  string `json:"device_type"`
}

type geoSubdivision struct {
	Names map[string]string `maxminddb:"names"`
}

type accessSourceEvent struct {
	Port      int
	IP        string
	Timestamp int64
}

type accessSourceState struct {
	mu       sync.Mutex
	logPath  string
	offset   int64
	lastRun  time.Time
	ports    map[int]*portUsageAccumulator
	report   *AccessSourceReport
	resolver ipProfileResolver
}

type portUsageAccumulator struct {
	stats *PortUsageStat
	ips   map[string]*IPUsageStat
}

type ipProfileResolver interface {
	Resolve(ip string) IPProfile
}

type cachedIPProfile struct {
	profile   IPProfile
	expiresAt time.Time
}

type cachingIPProfileResolver struct {
	mu       sync.Mutex
	ttl      time.Duration
	now      func() time.Time
	lookup   func(string) IPProfile
	profiles map[string]cachedIPProfile
}

type maxMindIPProfileLookup struct {
	once   sync.Once
	cityDB *maxminddb.Reader
	asnDB  *maxminddb.Reader
}

var defaultAccessSourceState = newAccessSourceState()

func (s *AccessSourceService) Analyze() (*AccessSourceReport, error) {
	report := &AccessSourceReport{Ports: []*PortUsageStat{}}
	logPath, _ := s.accessLogPath()
	if strings.TrimSpace(logPath) == "" {
		report.Message = "未配置 access.log，端口共享检测暂无数据。"
		return report, nil
	}

	report, exists, err := defaultAccessSourceState.analyze(logPath)
	if err != nil {
		return report, err
	}
	if !exists {
		report.Message = "access.log 文件不存在，端口共享检测暂无数据。"
		return report, nil
	}
	if len(report.Ports) == 0 {
		report.Message = "access.log 中暂无可识别的入站访问记录。"
	}
	return report, nil
}

func newAccessSourceState() *accessSourceState {
	lookup := newMaxMindIPProfileLookup()
	return &accessSourceState{
		ports:    map[int]*portUsageAccumulator{},
		resolver: newCachingIPProfileResolver(ipProfileTTL, lookup.Lookup),
	}
}

func (s *accessSourceState) analyze(logPath string) (*AccessSourceReport, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.logPath != logPath {
		s.reset(logPath)
	}
	if s.report != nil && now.Sub(s.lastRun) < accessSourceBatchEvery {
		return cloneAccessSourceReport(s.report), true, nil
	}

	events, exists, err := s.readNewEvents(logPath)
	if err != nil || !exists {
		return &AccessSourceReport{Ports: []*PortUsageStat{}}, exists, err
	}
	for _, event := range events {
		s.applyEvent(event)
	}
	s.lastRun = now
	s.report = s.snapshot()
	return cloneAccessSourceReport(s.report), true, nil
}

func (s *accessSourceState) reset(logPath string) {
	s.logPath = logPath
	s.offset = 0
	s.lastRun = time.Time{}
	s.ports = map[int]*portUsageAccumulator{}
	s.report = nil
}

func (s *accessSourceState) readNewEvents(path string) ([]*accessSourceEvent, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, true, err
	}
	if info.Size() < s.offset {
		s.offset = 0
		s.ports = map[int]*portUsageAccumulator{}
		s.report = nil
	}
	if s.offset == info.Size() {
		return nil, true, nil
	}

	start := s.offset
	skipPartialLine := false
	if start == 0 && info.Size() > accessSourceMaxReadBytes {
		start = info.Size() - accessSourceMaxReadBytes
		skipPartialLine = true
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, true, err
	}

	reader := io.Reader(file)
	if skipPartialLine {
		buffered := bufio.NewReader(file)
		if _, err := buffered.ReadString('\n'); err != nil && err != io.EOF {
			return nil, true, err
		}
		reader = buffered
	}

	events, err := scanAccessSourceEvents(reader)
	if err != nil {
		return nil, true, err
	}
	s.offset = info.Size()
	return events, true, nil
}

func (s *accessSourceState) applyEvent(event *accessSourceEvent) {
	acc := s.ports[event.Port]
	if acc == nil {
		acc = &portUsageAccumulator{
			stats: &PortUsageStat{Port: event.Port},
			ips:   map[string]*IPUsageStat{},
		}
		s.ports[event.Port] = acc
	}
	acc.stats.TotalHits++
	if event.Timestamp > acc.stats.LastSeen {
		acc.stats.LastSeen = event.Timestamp
	}

	ipStats := acc.ips[event.IP]
	if ipStats == nil {
		ipStats = &IPUsageStat{IPProfile: s.resolver.Resolve(event.IP)}
		acc.ips[event.IP] = ipStats
	}
	ipStats.HitCount++
	if event.Timestamp > ipStats.LastSeen {
		ipStats.LastSeen = event.Timestamp
	}
}

func (s *accessSourceState) snapshot() *AccessSourceReport {
	report := &AccessSourceReport{Ports: make([]*PortUsageStat, 0, len(s.ports))}
	for _, acc := range s.ports {
		stats := &PortUsageStat{
			Port:      acc.stats.Port,
			UniqueIPs: len(acc.ips),
			TotalHits: acc.stats.TotalHits,
			LastSeen:  acc.stats.LastSeen,
			IPs:       make([]*IPUsageStat, 0, len(acc.ips)),
		}
		for _, ipStats := range acc.ips {
			stats.IPs = append(stats.IPs, cloneIPUsageStat(ipStats))
		}
		sort.Slice(stats.IPs, func(i, j int) bool {
			if stats.IPs[i].HitCount == stats.IPs[j].HitCount {
				return stats.IPs[i].IP < stats.IPs[j].IP
			}
			return stats.IPs[i].HitCount > stats.IPs[j].HitCount
		})
		report.Ports = append(report.Ports, stats)
	}
	sortPortUsageStats(report.Ports)
	return report
}

func (s *AccessSourceService) accessLogPath() (string, bool) {
	for _, path := range []string{xray.GetConfigPath(), ""} {
		if path == "" {
			template, err := s.settingService.GetXrayConfigTemplate()
			if err != nil {
				continue
			}
			if access := parseAccessLogPath([]byte(template)); access != "" {
				return access, true
			}
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil {
			if access := parseAccessLogPath(data); access != "" {
				return access, true
			}
		}
	}

	const fallback = "/var/log/xray/access.log"
	if _, err := os.Stat(fallback); err == nil {
		return fallback, false
	}
	return "", false
}

func parseAccessLogPath(data []byte) string {
	config := struct {
		Log struct {
			Access string `json:"access"`
		} `json:"log"`
	}{}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}
	return strings.TrimSpace(config.Log.Access)
}

func scanAccessSourceEvents(reader io.Reader) ([]*accessSourceEvent, error) {
	events := make([]*accessSourceEvent, 0)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		event, ok := parseAccessSourceLine(scanner.Text())
		if ok {
			events = append(events, event)
		}
	}
	return events, scanner.Err()
}

func parseAccessSourceLine(line string) (*accessSourceEvent, bool) {
	match := accessSourceEventPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return nil, false
	}
	seenAt, err := time.ParseInLocation(accessSourceTimeLayout, match[1], time.Local)
	if err != nil {
		return nil, false
	}
	sourceIP, err := sourceIPFromEndpoint(match[2])
	if err != nil || sourceIP == "" || isLoopbackIP(sourceIP) {
		return nil, false
	}
	port, err := strconv.Atoi(match[3])
	if err != nil || port <= 0 {
		return nil, false
	}
	return &accessSourceEvent{
		Port:      port,
		IP:        sourceIP,
		Timestamp: seenAt.Unix(),
	}, true
}

func sourceIPFromEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		idx := strings.LastIndex(endpoint, ":")
		if idx <= 0 || idx >= len(endpoint)-1 {
			return "", err
		}
		host = endpoint[:idx]
	}
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	for {
		parts := strings.SplitN(host, ":", 2)
		if len(parts) != 2 {
			break
		}
		prefix := strings.ToLower(strings.TrimSpace(parts[0]))
		if prefix != "tcp" && prefix != "udp" && prefix != "unix" {
			break
		}
		host = strings.TrimSpace(strings.Trim(parts[1], "[]"))
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", nil
	}
	return ip.String(), nil
}

func isLoopbackIP(raw string) bool {
	ip := net.ParseIP(raw)
	return ip != nil && ip.IsLoopback()
}

func buildPortUsageStats(events []*accessSourceEvent) []*PortUsageStat {
	state := &accessSourceState{
		ports:    map[int]*portUsageAccumulator{},
		resolver: staticIPProfileResolver{},
	}
	for _, event := range events {
		state.applyEvent(event)
	}
	return state.snapshot().Ports
}

func sortPortUsageStats(ports []*PortUsageStat) {
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].UniqueIPs == ports[j].UniqueIPs {
			if ports[i].TotalHits == ports[j].TotalHits {
				return ports[i].Port < ports[j].Port
			}
			return ports[i].TotalHits > ports[j].TotalHits
		}
		return ports[i].UniqueIPs > ports[j].UniqueIPs
	})
}

func cloneAccessSourceReport(report *AccessSourceReport) *AccessSourceReport {
	if report == nil {
		return &AccessSourceReport{Ports: []*PortUsageStat{}}
	}
	clone := &AccessSourceReport{
		Message: report.Message,
		Ports:   make([]*PortUsageStat, 0, len(report.Ports)),
	}
	for _, port := range report.Ports {
		next := &PortUsageStat{
			Port:      port.Port,
			UniqueIPs: port.UniqueIPs,
			TotalHits: port.TotalHits,
			LastSeen:  port.LastSeen,
			IPs:       make([]*IPUsageStat, 0, len(port.IPs)),
		}
		for _, ip := range port.IPs {
			next.IPs = append(next.IPs, cloneIPUsageStat(ip))
		}
		clone.Ports = append(clone.Ports, next)
	}
	return clone
}

func cloneIPUsageStat(stat *IPUsageStat) *IPUsageStat {
	if stat == nil {
		return nil
	}
	return &IPUsageStat{
		IPProfile: stat.IPProfile,
		HitCount:  stat.HitCount,
		LastSeen:  stat.LastSeen,
	}
}

type staticIPProfileResolver struct{}

func (staticIPProfileResolver) Resolve(ip string) IPProfile {
	return IPProfile{
		IP:         ip,
		Geo:        "未知",
		ISP:        "未知",
		DeviceType: "UNKNOWN",
	}
}

func newCachingIPProfileResolver(ttl time.Duration, lookup func(string) IPProfile) *cachingIPProfileResolver {
	return &cachingIPProfileResolver{
		ttl:      ttl,
		now:      time.Now,
		lookup:   lookup,
		profiles: map[string]cachedIPProfile{},
	}
}

func (r *cachingIPProfileResolver) Resolve(ip string) IPProfile {
	now := r.now()
	r.mu.Lock()
	cached, ok := r.profiles[ip]
	if ok && now.Before(cached.expiresAt) {
		r.mu.Unlock()
		return cached.profile
	}
	r.mu.Unlock()

	profile := r.lookup(ip)
	r.mu.Lock()
	r.profiles[ip] = cachedIPProfile{
		profile:   profile,
		expiresAt: now.Add(r.ttl),
	}
	r.mu.Unlock()
	return profile
}

func newMaxMindIPProfileLookup() *maxMindIPProfileLookup {
	return &maxMindIPProfileLookup{}
}

func (l *maxMindIPProfileLookup) Lookup(rawIP string) IPProfile {
	l.once.Do(l.open)
	profile := staticIPProfileResolver{}.Resolve(rawIP)
	ip := net.ParseIP(rawIP)
	if ip == nil {
		return profile
	}

	if l.cityDB != nil {
		var city struct {
			Country struct {
				ISOCode string            `maxminddb:"iso_code"`
				Names   map[string]string `maxminddb:"names"`
			} `maxminddb:"country"`
			Subdivisions []geoSubdivision `maxminddb:"subdivisions"`
			City         struct {
				Names map[string]string `maxminddb:"names"`
			} `maxminddb:"city"`
		}
		if err := l.cityDB.Lookup(ip, &city); err == nil {
			profile.CountryCode = city.Country.ISOCode
			profile.City = preferredName(city.City.Names)
			profile.Geo = formatGeo(city.Country.ISOCode, city.Subdivisions, profile.City)
		}
	}

	var organization string
	if l.asnDB != nil {
		var asn struct {
			AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
			AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
		}
		if err := l.asnDB.Lookup(ip, &asn); err == nil {
			profile.ASN = asn.AutonomousSystemNumber
			organization = asn.AutonomousSystemOrganization
		}
	}
	profile.ISP = cleanISPName(organization)
	profile.DeviceType = inferDeviceType(organization)
	return profile
}

func (l *maxMindIPProfileLookup) open() {
	l.cityDB = openFirstMaxMindDB([]string{
		"/usr/share/GeoIP/GeoLite2-City.mmdb",
		"/usr/local/share/GeoIP/GeoLite2-City.mmdb",
		"/var/lib/GeoIP/GeoLite2-City.mmdb",
		"./GeoLite2-City.mmdb",
	})
	l.asnDB = openFirstMaxMindDB([]string{
		"/usr/share/GeoIP/GeoLite2-ASN.mmdb",
		"/usr/local/share/GeoIP/GeoLite2-ASN.mmdb",
		"/var/lib/GeoIP/GeoLite2-ASN.mmdb",
		"./GeoLite2-ASN.mmdb",
	})
}

func openFirstMaxMindDB(paths []string) *maxminddb.Reader {
	for _, path := range paths {
		reader, err := maxminddb.Open(path)
		if err == nil {
			return reader
		}
	}
	return nil
}

func preferredName(names map[string]string) string {
	if names == nil {
		return ""
	}
	for _, key := range []string{"zh-CN", "en"} {
		if value := strings.TrimSpace(names[key]); value != "" {
			return value
		}
	}
	for _, value := range names {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func formatGeo(country string, subdivisions []geoSubdivision, city string) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(country) != "" {
		parts = append(parts, strings.TrimSpace(country))
	}
	if len(subdivisions) > 0 {
		if subdivision := preferredName(subdivisions[0].Names); subdivision != "" {
			parts = append(parts, subdivision)
		}
	}
	if strings.TrimSpace(city) != "" {
		parts = append(parts, strings.TrimSpace(city))
	}
	if len(parts) == 0 {
		return "未知"
	}
	return strings.Join(parts, "/")
}

func cleanISPName(organization string) string {
	lower := strings.ToLower(organization)
	switch {
	case strings.Contains(lower, "china mobile"):
		return "China Mobile"
	case strings.Contains(lower, "china telecom"):
		return "China Telecom"
	case strings.Contains(lower, "china unicom"):
		return "China Unicom"
	case strings.Contains(lower, "amazon") || strings.Contains(lower, "aws"):
		return "AWS"
	case strings.Contains(lower, "google"):
		return "Google"
	case strings.Contains(lower, "oracle"):
		return "Oracle"
	case strings.Contains(lower, "alibaba") || strings.Contains(lower, "aliyun"):
		return "Alibaba"
	case strings.TrimSpace(organization) == "":
		return "未知"
	default:
		fields := strings.Fields(organization)
		if len(fields) > 3 {
			fields = fields[:3]
		}
		return strings.Join(fields, " ")
	}
}

func inferDeviceType(organization string) string {
	lower := strings.ToLower(organization)
	switch {
	case strings.Contains(lower, "amazon") || strings.Contains(lower, "aws") ||
		strings.Contains(lower, "google") || strings.Contains(lower, "oracle") ||
		strings.Contains(lower, "alibaba") || strings.Contains(lower, "aliyun"):
		return "CLOUD_SERVER"
	case strings.Contains(lower, "mobile") || strings.Contains(lower, "4g") || strings.Contains(lower, "5g"):
		return "MOBILE"
	case strings.Contains(lower, "china telecom") || strings.Contains(lower, "china unicom"):
		return "HOME_BROADBAND"
	case strings.Contains(lower, "hosting") || strings.Contains(lower, "vps") || strings.Contains(lower, "server"):
		return "DATACENTER"
	default:
		return "UNKNOWN"
	}
}
