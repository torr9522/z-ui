package service

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/atomic"
	"sync"
	"time"
	xruntime "x-ui/runtime"
	"x-ui/xray"
)

var isNeedXrayRestart atomic.Bool
var runtimeManagerOnce sync.Once
var runtimeManager *xruntime.Manager
var runtimeReconciler *xruntime.Reconciler

type XrayService struct {
	inboundService InboundService
	settingService SettingService
}

func (s *XrayService) getManager() *xruntime.Manager {
	runtimeManagerOnce.Do(func() {
		runtimeManager = xruntime.NewManager(&s.settingService)
		runtimeReconciler = xruntime.NewReconciler(runtimeManager, s)
	})
	return runtimeManager
}

func (s *XrayService) getReconciler() *xruntime.Reconciler {
	s.getManager()
	return runtimeReconciler
}

func (s *XrayService) IsXrayRunning() bool {
	return s.getManager().IsRunning()
}

func (s *XrayService) GetXrayErr() error {
	return s.getManager().GetErr()
}

func (s *XrayService) GetXrayResult() string {
	return s.getManager().GetResult()
}

func (s *XrayService) GetXrayVersion() string {
	return s.getManager().GetVersion()
}

func (s *XrayService) BuildRuntimeSnapshot() (*xruntime.Snapshot, error) {
	templateConfig, err := s.settingService.GetXrayConfigTemplate()
	if err != nil {
		return nil, err
	}
	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}
	return s.getManager().Builder().Build(templateConfig, inbounds)
}

func (s *XrayService) GetXrayConfig() (*xray.Config, error) {
	snapshot, err := s.BuildRuntimeSnapshot()
	if err != nil {
		return nil, err
	}
	return snapshot.Raw, nil
}

func (s *XrayService) GetXrayTraffic() ([]*xray.Traffic, error) {
	if !s.IsXrayRunning() {
		return nil, errors.New("xray is not running")
	}
	return s.getManager().GetTraffic(true)
}

func (s *XrayService) RestartXray(isForce bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err := s.getReconciler().Reconcile(ctx, isForce)
	if err != nil {
		return fmt.Errorf("sync xray runtime failed: %w", err)
	}
	return nil
}

func (s *XrayService) StopXray() error {
	return s.getManager().Stop()
}

func (s *XrayService) RecoverRuntime() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return s.getReconciler().RecoverIfDirty(ctx)
}

func (s *XrayService) CheckRuntimeHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.getReconciler().HealthCheck(ctx)
}

func (s *XrayService) SetToNeedRestart() {
	isNeedXrayRestart.Store(true)
}

func (s *XrayService) IsNeedRestartAndSetFalse() bool {
	return isNeedXrayRestart.CAS(true, false)
}
