package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"x-ui/logger"
	"x-ui/xray"
)

type StatusStore interface {
	SetRuntimeDirty(bool) error
	SetRuntimeLastError(string) error
	GetRuntimeDirty() (bool, error)
}

type Manager struct {
	mu       sync.Mutex
	process  *xray.Process
	snapshot *Snapshot
	result   string

	builder  Builder
	planner  Planner
	settings StatusStore
}

func NewManager(settings StatusStore) *Manager {
	return &Manager{
		settings: settings,
	}
}

func (m *Manager) Builder() *Builder {
	return &m.builder
}

func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunningLocked()
}

func (m *Manager) GetErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process == nil {
		return nil
	}
	return m.process.GetErr()
}

func (m *Manager) GetResult() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.result != "" {
		return m.result
	}
	if m.process == nil || m.isRunningLocked() {
		return ""
	}
	m.result = m.process.GetResult()
	return m.result
}

func (m *Manager) GetVersion() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process == nil {
		return "Unknown"
	}
	return m.process.GetVersion()
}

func (m *Manager) GetTraffic(reset bool) ([]*xray.Traffic, error) {
	m.mu.Lock()
	process := m.process
	m.mu.Unlock()

	if process == nil || !process.IsRunning() {
		return nil, errors.New("xray is not running")
	}
	return process.GetTraffic(reset)
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.Debug("stop xray")
	if !m.isRunningLocked() {
		return errors.New("xray is not running")
	}
	return m.process.Stop()
}

func (m *Manager) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	process := m.process
	m.mu.Unlock()

	if process == nil || !process.IsRunning() {
		return errors.New("xray is not running")
	}
	return NewClient(process.GetAPIPort()).HealthCheck(ctx)
}

func (m *Manager) Sync(ctx context.Context, desired *Snapshot, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if desired == nil {
		return errors.New("desired runtime snapshot is nil")
	}

	if !force && m.isRunningLocked() && m.snapshot != nil && m.snapshot.Raw.Equals(desired.Raw) {
		if err := m.healthCheckLocked(ctx); err == nil {
			return m.clearRuntimeStatusLocked()
		}
		return m.fallbackFullRestartLocked(ctx, desired, errors.New("runtime health check failed on unchanged config"))
	}

	if force || !m.isRunningLocked() || m.snapshot == nil {
		return m.fullRestartLocked(desired)
	}

	plan, err := m.planner.Plan(m.snapshot, desired)
	if err != nil {
		return m.fallbackFullRestartLocked(ctx, desired, err)
	}
	if plan.NeedFullSync {
		return m.fallbackFullRestartLocked(ctx, desired, errors.New(plan.Reason))
	}
	if len(plan.Operations) == 0 {
		return m.clearRuntimeStatusLocked()
	}

	if err := m.healthCheckLocked(ctx); err != nil {
		return m.fallbackFullRestartLocked(ctx, desired, err)
	}

	if err := NewClient(m.process.GetAPIPort()).Apply(ctx, plan); err != nil {
		return m.fallbackFullRestartLocked(ctx, desired, err)
	}

	m.snapshot = desired
	return m.clearRuntimeStatusLocked()
}

func (m *Manager) fullRestartLocked(desired *Snapshot) error {
	logger.Debug("restart xray with full sync")
	if m.isRunningLocked() {
		_ = m.process.Stop()
	}

	process := xray.NewProcess(desired.Raw)
	m.result = ""
	if err := process.Start(); err != nil {
		_ = m.markRuntimeIssueLocked(fmt.Errorf("full restart failed: %w", err))
		m.process = process
		return err
	}

	m.process = process
	m.snapshot = desired

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.waitForHealthyLocked(ctx); err != nil {
		_ = m.markRuntimeIssueLocked(fmt.Errorf("runtime api health check failed after restart: %w", err))
		return err
	}

	return m.clearRuntimeStatusLocked()
}

func (m *Manager) fallbackFullRestartLocked(ctx context.Context, desired *Snapshot, cause error) error {
	if cause != nil {
		logger.Warning("runtime api apply failed, fallback to full restart:", cause)
		if err := m.markRuntimeIssueLocked(cause); err != nil {
			logger.Warning("persist runtime failure failed:", err)
		}
	}
	if err := m.fullRestartLocked(desired); err != nil {
		return err
	}
	return nil
}

func (m *Manager) healthCheckLocked(ctx context.Context) error {
	if !m.isRunningLocked() {
		return errors.New("xray is not running")
	}
	return NewClient(m.process.GetAPIPort()).HealthCheck(ctx)
}

func (m *Manager) waitForHealthyLocked(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := m.healthCheckLocked(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) isRunningLocked() bool {
	return m.process != nil && m.process.IsRunning()
}

func (m *Manager) markRuntimeIssueLocked(err error) error {
	if m.settings == nil {
		return nil
	}
	if setErr := m.settings.SetRuntimeDirty(true); setErr != nil {
		return setErr
	}
	return m.settings.SetRuntimeLastError(err.Error())
}

func (m *Manager) clearRuntimeStatusLocked() error {
	if m.settings == nil {
		return nil
	}
	if err := m.settings.SetRuntimeDirty(false); err != nil {
		return err
	}
	return m.settings.SetRuntimeLastError("")
}
