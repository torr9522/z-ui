package runtime

import (
	"context"
)

type SnapshotBuilder interface {
	BuildRuntimeSnapshot() (*Snapshot, error)
}

type Reconciler struct {
	manager *Manager
	builder SnapshotBuilder
}

func NewReconciler(manager *Manager, builder SnapshotBuilder) *Reconciler {
	return &Reconciler{
		manager: manager,
		builder: builder,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, force bool) error {
	snapshot, err := r.builder.BuildRuntimeSnapshot()
	if err != nil {
		return err
	}
	return r.manager.Sync(ctx, snapshot, force)
}

func (r *Reconciler) HealthCheck(ctx context.Context) error {
	return r.manager.HealthCheck(ctx)
}

func (r *Reconciler) RecoverIfDirty(ctx context.Context) error {
	dirty, err := r.manager.settings.GetRuntimeDirty()
	if err != nil {
		return err
	}
	if !dirty {
		return r.manager.HealthCheck(ctx)
	}
	return r.Reconcile(ctx, true)
}
