# Runtime System

The runtime system turns DB state into live Xray state.

## Components

- `runtime.Builder`: builds desired Xray snapshot from template + inbounds.
- `runtime.Planner`: decides whether Runtime API can apply a diff.
- `runtime.Client`: wraps Xray Runtime API calls.
- `runtime.Manager`: applies plans, health checks, fallback restarts.
- `runtime.Reconciler`: orchestration and dirty-state recovery.

## Principle

Database is target state. Runtime is materialized state.

## Fallback

If Runtime API is unavailable, unhealthy, or cannot safely apply the diff, the manager falls back to full Xray restart and records runtime error state.
