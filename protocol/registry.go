package protocol

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	modules map[string]Module
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *Registry
)

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		registry := NewRegistry()
		registry.Register(newVMessModule())
		registry.Register(newVLESSModule())
		registry.Register(newTrojanModule())
		registry.Register(newShadowsocksModule())
		registry.Register(newMixedModule())
		registry.Register(newPassthroughModule("dokodemo-door", false))
		registry.Register(newPassthroughModule("http", false))
		registry.Register(newPassthroughModule("socks", false))
		defaultRegistry = registry
	})
	return defaultRegistry
}

func (r *Registry) Register(module Module) {
	r.modules[strings.ToLower(module.Name())] = module
}

func (r *Registry) Get(name string) (Module, error) {
	module, ok := r.modules[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("protocol module %q is not registered", name)
	}
	return module, nil
}
