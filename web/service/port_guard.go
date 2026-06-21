package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"x-ui/database"
	"x-ui/database/model"
)

const (
	portGuardSyncScript = "/usr/local/bin/zui-port-guard-sync"
	portGuardLogPath    = "/var/log/z-ui/port-guard.log"
	portGuardTableName  = "zui_port_guard"
)

type PortGuardService struct{}

type PortGuardStatus struct {
	Ports          []PortGuardPortStatus `json:"ports"`
	WhitelistPorts []int                 `json:"whitelistPorts"`
	NftTableReady  bool                  `json:"nftTableReady"`
	TimerActive    bool                  `json:"timerActive"`
	LastSyncAt     int64                 `json:"lastSyncAt"`
}

type PortGuardPortStatus struct {
	Port                 int    `json:"port"`
	Protocol             string `json:"protocol"`
	Remark               string `json:"remark"`
	Enabled              bool   `json:"enabled"`
	WindowSeconds        int    `json:"windowSeconds"`
	IPLimit              int    `json:"ipLimit"`
	BanSeconds           int    `json:"banSeconds"`
	IPv4Count            int    `json:"ipv4Count"`
	State                string `json:"state"`
	Banned               bool   `json:"banned"`
	BannedUntil          int64  `json:"bannedUntil"`
	RemainingSeconds     int64  `json:"remainingSeconds"`
	LastTriggerIP        string `json:"lastTriggerIp"`
	LastTriggerAt        int64  `json:"lastTriggerAt"`
	ProtectedByWhitelist bool   `json:"protectedByWhitelist"`
}

type PortGuardSyncResult struct {
	Applied      bool               `json:"applied"`
	ManagedPorts []int              `json:"managedPorts"`
	SkippedPorts []SkippedPortGuard `json:"skippedPorts"`
	Output       string             `json:"output"`
}

type SkippedPortGuard struct {
	Port   int    `json:"port"`
	Reason string `json:"reason"`
}

type PortGuardUnbanRequest struct {
	Port int `json:"port" form:"port"`
}

func (s *PortGuardService) SyncNow() {
	go func() {
		_, _ = s.SyncNowBlocking()
	}()
}

func (s *PortGuardService) SyncNowBlocking() (*PortGuardSyncResult, error) {
	output, err := runPortGuardCommand(20*time.Second, portGuardSyncScript)
	result := parseSyncOutput(output)
	result.Output = output
	return result, err
}

func (s *PortGuardService) Status() (*PortGuardStatus, error) {
	inbounds, err := s.loadInbounds()
	if err != nil {
		return nil, err
	}
	whitelist, err := s.WhitelistPorts()
	if err != nil {
		return nil, err
	}
	whitelistMap := intSet(whitelist)
	tableReady := nftTableReady()
	blocked := map[int]int64{}
	counts := map[int]int{}
	if tableReady {
		blocked = nftBlockedPorts()
		counts = nftPortIPv4Counts(inbounds)
	}

	now := time.Now().Unix()
	ports := make([]PortGuardPortStatus, 0, len(inbounds))
	for _, inbound := range inbounds {
		enabled := inbound.PortGuardEnabled && inbound.PortGuardIPCount > 0
		protected := whitelistMap[inbound.Port]
		bannedUntil := inbound.PortGuardBannedUntil
		remaining := int64(0)
		if timeout, ok := blocked[inbound.Port]; ok {
			if timeout > 0 {
				bannedUntil = now + timeout
				remaining = timeout
			} else {
				remaining = maxInt64(0, bannedUntil-now)
			}
		} else if bannedUntil > now {
			remaining = bannedUntil - now
		} else {
			bannedUntil = 0
		}
		state := "OFF"
		if protected {
			state = "WHITELISTED"
		} else if enabled {
			state = "ACTIVE"
		}
		if !tableReady && enabled && !protected {
			state = "ERROR"
		}
		if remaining > 0 && enabled && !protected {
			state = "BANNED"
		}
		ports = append(ports, PortGuardPortStatus{
			Port:                 inbound.Port,
			Protocol:             string(inbound.Protocol),
			Remark:               inbound.Remark,
			Enabled:              enabled,
			WindowSeconds:        inbound.PortGuardWindowSeconds,
			IPLimit:              inbound.PortGuardIPCount,
			BanSeconds:           inbound.PortGuardBanSeconds,
			IPv4Count:            counts[inbound.Port],
			State:                state,
			Banned:               state == "BANNED",
			BannedUntil:          bannedUntil,
			RemainingSeconds:     remaining,
			LastTriggerIP:        inbound.PortGuardLastTriggerIP,
			LastTriggerAt:        inbound.PortGuardLastTriggerAt,
			ProtectedByWhitelist: protected,
		})
	}
	return &PortGuardStatus{
		Ports:          ports,
		WhitelistPorts: whitelist,
		NftTableReady:  tableReady,
		TimerActive:    timerActive("zui-port-guard-sync.timer"),
		LastSyncAt:     lastSyncAt(),
	}, nil
}

func (s *PortGuardService) Unban(port int, operator string) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	if nftTableReady() {
		_, _ = runPortGuardCommand(8*time.Second, "nft", "delete", "element", "inet", portGuardTableName, "blocked_ports", "{", strconv.Itoa(port), "}")
		_, _ = runPortGuardCommand(8*time.Second, "nft", "flush", "set", "inet", portGuardTableName, fmt.Sprintf("pg4_%d", port))
	}
	db := database.GetDB()
	if err := db.Model(&model.Inbound{}).Where("port = ?", port).Updates(map[string]interface{}{
		"port_guard_banned_until":    0,
		"port_guard_last_trigger_ip": "",
		"port_guard_last_trigger_at": 0,
	}).Error; err != nil {
		return err
	}
	appendPortGuardLog(map[string]interface{}{
		"time":     time.Now().UTC().Format(time.RFC3339),
		"event":    "manual_unban",
		"port":     port,
		"operator": operator,
	})
	return nil
}

func (s *PortGuardService) Logs(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	file, err := os.Open(portGuardLogPath)
	if os.IsNotExist(err) {
		return []map[string]interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	records := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		record := map[string]interface{}{}
		if err := json.Unmarshal([]byte(line), &record); err == nil {
			records = append(records, record)
		}
	}
	return records, nil
}

func (s *PortGuardService) WhitelistPorts() ([]int, error) {
	ports := map[int]bool{22: true, 80: true, 443: true}
	if sshPorts, err := sshConfigPorts("/etc/ssh/sshd_config"); err == nil {
		for _, port := range sshPorts {
			ports[port] = true
		}
	}
	settingService := SettingService{}
	if port, err := settingService.GetPort(); err == nil && port > 0 {
		ports[port] = true
	}
	raw, err := settingService.getString("portGuardWhitelistPorts")
	if err == nil {
		for _, port := range parseWhitelistPorts(raw) {
			ports[port] = true
		}
	}
	result := make([]int, 0, len(ports))
	for port := range ports {
		if port > 0 && port <= 65535 {
			result = append(result, port)
		}
	}
	sort.Ints(result)
	return result, nil
}

func (s *PortGuardService) loadInbounds() ([]model.Inbound, error) {
	db := database.GetDB()
	var inbounds []model.Inbound
	err := db.Order("port asc").Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func runPortGuardCommand(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%s failed: %w", name, err)
	}
	return out.String(), nil
}

func nftTableReady() bool {
	_, err := runPortGuardCommand(4*time.Second, "nft", "list", "table", "inet", portGuardTableName)
	return err == nil
}

func nftBlockedPorts() map[int]int64 {
	out, err := runPortGuardCommand(4*time.Second, "nft", "list", "set", "inet", portGuardTableName, "blocked_ports")
	if err != nil {
		return map[int]int64{}
	}
	return parseNftBlockedPorts(out)
}

func parseNftBlockedPorts(out string) map[int]int64 {
	result := map[int]int64{}
	body := nftElementsBody(out)
	if body == "" {
		return result
	}
	re := regexp.MustCompile(`^\s*([0-9]{1,5})\b`)
	for _, item := range strings.Split(body, ",") {
		match := re.FindStringSubmatch(item)
		if match == nil {
			continue
		}
		port, _ := strconv.Atoi(match[1])
		if port <= 0 || port > 65535 {
			continue
		}
		result[port] = parseNftElementDuration(item)
	}
	return result
}

func nftPortIPv4Counts(inbounds []model.Inbound) map[int]int {
	result := map[int]int{}
	for _, inbound := range inbounds {
		if !inbound.PortGuardEnabled || inbound.PortGuardIPCount <= 0 {
			continue
		}
		out, err := runPortGuardCommand(4*time.Second, "nft", "list", "set", "inet", portGuardTableName, fmt.Sprintf("pg4_%d", inbound.Port))
		if err != nil {
			continue
		}
		result[inbound.Port] = countIPv4Elements(out)
	}
	return result
}

func countIPv4Elements(out string) int {
	re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	seen := map[string]bool{}
	for _, value := range re.FindAllString(out, -1) {
		seen[value] = true
	}
	return len(seen)
}

func nftElementsBody(out string) string {
	match := regexp.MustCompile(`(?s)elements\s*=\s*\{(.*?)\}`).FindStringSubmatch(out)
	if match == nil {
		return ""
	}
	return match[1]
}

func parseNftElementDuration(item string) int64 {
	if match := regexp.MustCompile(`\bexpires\s+([0-9smhd]+)`).FindStringSubmatch(item); match != nil {
		return parseNftDuration(match[1])
	}
	if match := regexp.MustCompile(`\btimeout\s+([0-9smhd]+)`).FindStringSubmatch(item); match != nil {
		return parseNftDuration(match[1])
	}
	return 0
}

func parseNftDuration(value string) int64 {
	if value == "" {
		return 0
	}
	total := int64(0)
	re := regexp.MustCompile(`([0-9]+)([smhd])`)
	for _, match := range re.FindAllStringSubmatch(value, -1) {
		n, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		switch match[2] {
		case "m":
			total += n * 60
		case "h":
			total += n * 3600
		case "d":
			total += n * 86400
		default:
			total += n
		}
	}
	return total
}

func timerActive(name string) bool {
	out, err := runPortGuardCommand(4*time.Second, "systemctl", "is-active", name)
	return err == nil && strings.TrimSpace(out) == "active"
}

func lastSyncAt() int64 {
	info, err := os.Stat("/var/lib/z-ui/port-guard/desired.conf")
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

func parseSyncOutput(output string) *PortGuardSyncResult {
	result := &PortGuardSyncResult{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "applied" || line == "unchanged" {
			result.Applied = true
			continue
		}
		if strings.HasPrefix(line, "managed_ports=") {
			result.ManagedPorts = parseCSVPorts(strings.TrimPrefix(line, "managed_ports="))
			continue
		}
		if strings.HasPrefix(line, "skipped_ports=") {
			result.SkippedPorts = parseSkippedPorts(strings.TrimPrefix(line, "skipped_ports="))
		}
	}
	return result
}

func parseCSVPorts(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return []int{}
	}
	ports := make([]int, 0)
	for _, item := range regexp.MustCompile(`[\s,]+`).Split(raw, -1) {
		port, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && port > 0 && port <= 65535 {
			ports = append(ports, port)
		}
	}
	return ports
}

func parseSkippedPorts(raw string) []SkippedPortGuard {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return []SkippedPortGuard{}
	}
	items := make([]SkippedPortGuard, 0)
	for _, item := range strings.Split(raw, ",") {
		parts := strings.SplitN(item, ":", 2)
		port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		reason := "skipped"
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			reason = strings.TrimSpace(parts[1])
		}
		items = append(items, SkippedPortGuard{Port: port, Reason: reason})
	}
	return items
}

func parseWhitelistPorts(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}
	}
	ports := make([]int, 0)
	var obj struct {
		Ports []int `json:"ports"`
	}
	if strings.HasPrefix(raw, "{") {
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for _, port := range obj.Ports {
				if port > 0 && port <= 65535 {
					ports = append(ports, port)
				}
			}
			return ports
		}
	}
	var arr []int
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			for _, port := range arr {
				if port > 0 && port <= 65535 {
					ports = append(ports, port)
				}
			}
			return ports
		}
	}
	for _, item := range regexp.MustCompile(`[\s,]+`).Split(raw, -1) {
		port, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && port > 0 && port <= 65535 {
			ports = append(ports, port)
		}
	}
	return ports
}

func sshConfigPorts(path string) ([]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ports := make([]int, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.ToLower(fields[0]) != "port" {
			continue
		}
		port, err := strconv.Atoi(fields[1])
		if err == nil && port > 0 && port <= 65535 {
			ports = append(ports, port)
		}
	}
	return ports, scanner.Err()
}

func intSet(values []int) map[int]bool {
	result := make(map[int]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func appendPortGuardLog(record map[string]interface{}) {
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = os.MkdirAll("/var/log/z-ui", 0755)
	file, err := os.OpenFile(portGuardLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
