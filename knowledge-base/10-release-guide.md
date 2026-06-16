# Release Guide

## Pre-release Checklist

1. Worktree has no unintended tracked changes.
2. No sensitive files are tracked.
3. `go build ./...` passes.
4. `go test ./...` passes.
5. Smoke tests pass.
6. DOM checks pass when UI changed.
7. Install smoke passes when installer changed.
8. Archive/combined backup generated outside tracked files.

## Suggested Tag Flow

Do not tag automatically. For this repository-ready state, suggested tag:

`beta-v0.1.6-repository-ready`
