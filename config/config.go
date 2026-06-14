package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"
)

//go:embed version
var version string

//go:embed name
var name string

var assetVersion = resolveAssetVersion()

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func GetVersion() string {
	return strings.TrimSpace(version)
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
	return fmt.Sprintf("/etc/%s/%s.db", GetName(), GetName())
}

func resolveAssetVersion() string {
	if value := strings.TrimSpace(os.Getenv("XUI_ASSET_VERSION")); value != "" {
		return value
	}
	return fmt.Sprintf("%s-%d", GetVersion(), time.Now().Unix())
}
