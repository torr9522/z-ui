package model

import (
	"fmt"
	"time"
	"x-ui/util/json_util"
	"x-ui/xray"
)

type Protocol string

const (
	VMess       Protocol = "vmess"
	VLESS       Protocol = "vless"
	Dokodemo    Protocol = "dokodemo-door"
	Socks       Protocol = "socks"
	Http        Protocol = "http"
	Mixed       Protocol = "mixed"
	Tunnel      Protocol = "tunnel"
	Trojan      Protocol = "trojan"
	Shadowsocks Protocol = "shadowsocks"
)

type User struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	PasswordHash string `json:"-" gorm:"column:password_hash"`
}

type Inbound struct {
	Id         int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int    `json:"-"`
	Up         int64  `json:"up" form:"up"`
	Down       int64  `json:"down" form:"down"`
	Total      int64  `json:"total" form:"total"`
	Remark     string `json:"remark" form:"remark"`
	Enable     bool   `json:"enable" form:"enable"`
	ExpiryTime int64  `json:"expiryTime" form:"expiryTime"`

	PortGuardEnabled       bool   `json:"portGuardEnabled" form:"portGuardEnabled" gorm:"column:port_guard_enabled;default:false"`
	PortGuardWindowSeconds int    `json:"portGuardWindowSeconds" form:"portGuardWindowSeconds" gorm:"column:port_guard_window_seconds;default:300"`
	PortGuardIPCount       int    `json:"portGuardIpCount" form:"portGuardIpCount" gorm:"column:port_guard_ip_count;default:0"`
	PortGuardBanSeconds    int    `json:"portGuardBanSeconds" form:"portGuardBanSeconds" gorm:"column:port_guard_ban_seconds;default:300"`
	PortGuardBannedUntil   int64  `json:"portGuardBannedUntil" form:"portGuardBannedUntil" gorm:"column:port_guard_banned_until;default:0"`
	PortGuardLastTriggerIP string `json:"portGuardLastTriggerIp" form:"portGuardLastTriggerIp" gorm:"column:port_guard_last_trigger_ip;default:''"`
	PortGuardLastTriggerAt int64  `json:"portGuardLastTriggerAt" form:"portGuardLastTriggerAt" gorm:"column:port_guard_last_trigger_at;default:0"`

	// config part
	Listen         string   `json:"listen" form:"listen"`
	Port           int      `json:"port" form:"port" gorm:"unique"`
	Protocol       Protocol `json:"protocol" form:"protocol"`
	Settings       string   `json:"settings" form:"settings"`
	StreamSettings string   `json:"streamSettings" form:"streamSettings"`
	Tag            string   `json:"tag" form:"tag" gorm:"unique"`
	Sniffing       string   `json:"sniffing" form:"sniffing"`
}

func (i *Inbound) GenXrayInboundConfig() *xray.InboundConfig {
	listen := i.Listen
	if listen != "" {
		listen = fmt.Sprintf("\"%v\"", listen)
	}
	return &xray.InboundConfig{
		Listen:         json_util.RawMessage(listen),
		Port:           i.Port,
		Protocol:       string(i.Protocol),
		Settings:       json_util.RawMessage(i.Settings),
		StreamSettings: json_util.RawMessage(i.StreamSettings),
		Tag:            i.Tag,
		Sniffing:       json_util.RawMessage(i.Sniffing),
	}
}

type Setting struct {
	Id    int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key" form:"key"`
	Value string `json:"value" form:"value"`
}

type SchemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"autoCreateTime"`
}

type InboundClient struct {
	Id        int       `gorm:"primaryKey;autoIncrement"`
	InboundId int       `gorm:"index;not null"`
	Email     string    `gorm:"index"`
	ClientKey string    `gorm:"index"`
	Settings  string    `gorm:"not null"`
	Enable    bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
