package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	commonprotocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	xtcore "github.com/xtls/xray-core/core"
	xtserial "github.com/xtls/xray-core/infra/conf/serial"
	xtshadowsocks "github.com/xtls/xray-core/proxy/shadowsocks"
	xtshadowsocks2022 "github.com/xtls/xray-core/proxy/shadowsocks_2022"
	xttrojan "github.com/xtls/xray-core/proxy/trojan"
	xtvlessinbound "github.com/xtls/xray-core/proxy/vless/inbound"
	xtvmessinbound "github.com/xtls/xray-core/proxy/vmess/inbound"
	"google.golang.org/protobuf/proto"
	"x-ui/database/model"
	xprotocol "x-ui/protocol"
	"x-ui/util/common"
	"x-ui/xray"
)

type Builder struct {
	registry *xprotocol.Registry
}

type Snapshot struct {
	Raw      *xray.Config
	Core     *xtcore.Config
	Inbounds map[string]*InboundState
}

type InboundState struct {
	DB         *model.Inbound
	Raw        *xray.InboundConfig
	Core       *xtcore.InboundHandlerConfig
	Comparable *xtcore.InboundHandlerConfig
	Users      map[string]*commonprotocol.User
}

func (b *Builder) Build(templateConfig string, inbounds []*model.Inbound) (*Snapshot, error) {
	if assetDir, err := filepath.Abs("bin"); err == nil {
		_ = os.Setenv("XRAY_LOCATION_ASSET", assetDir)
		_ = os.Setenv("xray.location.asset", assetDir)
	}

	rawConfig := &xray.Config{}
	if err := json.Unmarshal([]byte(templateConfig), rawConfig); err != nil {
		return nil, err
	}

	enabledTags := make([]string, 0)
	registry := b.registry
	if registry == nil {
		registry = xprotocol.DefaultRegistry()
	}
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		module, err := registry.Get(string(inbound.Protocol))
		if err != nil {
			return nil, err
		}
		inboundConfig, err := module.BuildInbound(inbound)
		if err != nil {
			return nil, err
		}
		rawConfig.InboundConfigs = append(rawConfig.InboundConfigs, *inboundConfig)
		enabledTags = append(enabledTags, inbound.Tag)
	}

	data, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, common.NewError("marshal xray config failed:", err)
	}

	coreConfig, err := xtserial.LoadJSONConfig(bytes.NewReader(data))
	if err != nil {
		return nil, common.NewError("build xray core config failed:", err)
	}

	states := make(map[string]*InboundState, len(enabledTags))
	coreInbounds := make(map[string]*xtcore.InboundHandlerConfig, len(coreConfig.Inbound))
	for _, inbound := range coreConfig.Inbound {
		coreInbounds[inbound.Tag] = inbound
	}

	rawInbounds := make(map[string]*xray.InboundConfig, len(rawConfig.InboundConfigs))
	for i := range rawConfig.InboundConfigs {
		inbound := &rawConfig.InboundConfigs[i]
		rawInbounds[inbound.Tag] = inbound
	}

	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		coreInbound, ok := coreInbounds[inbound.Tag]
		if !ok {
			return nil, fmt.Errorf("core inbound %q not found after build", inbound.Tag)
		}
		rawInbound, ok := rawInbounds[inbound.Tag]
		if !ok {
			return nil, fmt.Errorf("raw inbound %q not found after build", inbound.Tag)
		}

		comparable, users, err := splitInboundState(coreInbound)
		if err != nil {
			return nil, err
		}
		states[inbound.Tag] = &InboundState{
			DB:         inbound,
			Raw:        rawInbound,
			Core:       coreInbound,
			Comparable: comparable,
			Users:      users,
		}
	}

	return &Snapshot{
		Raw:      rawConfig,
		Core:     coreConfig,
		Inbounds: states,
	}, nil
}

func splitInboundState(inbound *xtcore.InboundHandlerConfig) (*xtcore.InboundHandlerConfig, map[string]*commonprotocol.User, error) {
	comparable := proto.Clone(inbound).(*xtcore.InboundHandlerConfig)
	users, stripped, err := splitUsersFromTypedMessage(comparable.ProxySettings)
	if err != nil {
		return nil, nil, err
	}
	comparable.ProxySettings = stripped
	return comparable, users, nil
}

func splitUsersFromTypedMessage(message *serial.TypedMessage) (map[string]*commonprotocol.User, *serial.TypedMessage, error) {
	if message == nil {
		return map[string]*commonprotocol.User{}, nil, nil
	}

	instance, err := message.GetInstance()
	if err != nil {
		return nil, nil, err
	}

	protoMessage, ok := instance.(proto.Message)
	if !ok {
		return nil, nil, fmt.Errorf("proxy settings %q is not a proto message", message.Type)
	}

	stripped := proto.Clone(protoMessage)
	users := make(map[string]*commonprotocol.User)

	switch cfg := stripped.(type) {
	case *xtvmessinbound.Config:
		for _, user := range cfg.User {
			if user == nil || user.Email == "" {
				continue
			}
			users[user.Email] = proto.Clone(user).(*commonprotocol.User)
		}
		cfg.User = nil
	case *xtvlessinbound.Config:
		for _, user := range cfg.Users {
			if user == nil || user.Email == "" {
				continue
			}
			users[user.Email] = proto.Clone(user).(*commonprotocol.User)
		}
		cfg.Users = nil
	case *xttrojan.ServerConfig:
		for _, user := range cfg.Users {
			if user == nil || user.Email == "" {
				continue
			}
			users[user.Email] = proto.Clone(user).(*commonprotocol.User)
		}
		cfg.Users = nil
	case *xtshadowsocks.ServerConfig:
		for _, user := range cfg.Users {
			if user == nil || user.Email == "" {
				continue
			}
			users[user.Email] = proto.Clone(user).(*commonprotocol.User)
		}
		cfg.Users = nil
	case *xtshadowsocks2022.MultiUserServerConfig:
		for _, user := range cfg.Users {
			if user == nil || user.Email == "" {
				continue
			}
			users[user.Email] = proto.Clone(user).(*commonprotocol.User)
		}
		cfg.Users = nil
	default:
		return map[string]*commonprotocol.User{}, serial.ToTypedMessage(protoMessage), nil
	}

	return users, serial.ToTypedMessage(stripped.(proto.Message)), nil
}
