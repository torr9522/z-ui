package config

import (
	_ "embed"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

//go:embed version
var version string

//go:embed name
var name string

var BuildCommit string
var BuildBranch string
var BuildTime string

var commit string
var branch string
var buildTime string

var versionMetadata = parseVersionMetadata(version)
var assetVersion = resolveAssetVersion()

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func GetVersion() string {
	if value := strings.TrimSpace(versionMetadata.Version); value != "" {
		return value
	}
	return strings.TrimSpace(version)
}

func GetBuildCommit() string {
	if value := normalizedBuildValue(BuildCommit); value != "未知" {
		return value
	}
	if value := normalizedBuildValue(commit); value != "未知" {
		return value
	}
	if value := normalizedBuildValue(versionMetadata.Commit); value != "未知" {
		if len(value) > 7 {
			return value[:7]
		}
		return value
	}
	if value := normalizedBuildValue(buildVCSRevision()); value != "未知" {
		if len(value) > 7 {
			return value[:7]
		}
		return value
	}
	return normalizedBuildValue(commit)
}

func GetBuildBranch() string {
	if value := normalizedBuildValue(BuildBranch); value != "未知" {
		return value
	}
	return normalizedBuildValue(branch)
}

func GetBuildTime() string {
	if value := normalizedBuildValue(BuildTime); value != "未知" {
		return value
	}
	if value := normalizedBuildValue(versionMetadata.BuildTime); value != "未知" {
		return value
	}
	return normalizedBuildValue(buildTime)
}

func GetAssetVersion() string {
	return assetVersion
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("XUI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool {
	return os.Getenv("XUI_DEBUG") == "true"
}

func GetDBPath() string {
	if value := strings.TrimSpace(os.Getenv("XUI_DB_PATH")); value != "" {
		return value
	}
	return fmt.Sprintf("/etc/%s/%s.db", GetName(), GetName())
}

func resolveAssetVersion() string {
	if value := strings.TrimSpace(os.Getenv("XUI_ASSET_VERSION")); value != "" {
		return value
	}
	return fmt.Sprintf("%s-%d", GetVersion(), time.Now().Unix())
}

func normalizedBuildValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "unknown" {
		return "未知"
	}
	return value
}

func buildVCSRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

type embeddedVersionMetadata struct {
	Version   string
	Commit    string
	BuildTime string
}

func parseVersionMetadata(raw string) embeddedVersionMetadata {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return embeddedVersionMetadata{}
	}
	if !strings.Contains(raw, "=") {
		return embeddedVersionMetadata{Version: raw}
	}

	metadata := embeddedVersionMetadata{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "version":
			metadata.Version = value
		case "commit":
			metadata.Commit = value
		case "build_time":
			metadata.BuildTime = value
		}
	}

	if metadata.Version == "" {
		metadata.Version = raw
	}
	return metadata
}
