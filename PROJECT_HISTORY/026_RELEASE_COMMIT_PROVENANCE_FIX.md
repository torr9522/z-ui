# 026 Release Commit Provenance Fix (v1.0.6)

## Problem

v1.0.5 was functionally usable, but release provenance was inconsistent:

- git tag v1.0.5 pointed to one commit
- config/version in the release package recorded an older commit
- the binary `x-ui version` output showed the same older embedded commit
- the remote installed commit therefore did not match the tag source commit

## Root Cause

- `config/version` was written before the final release commit existed.
- `scripts/build_release.sh` did not inject `BuildCommit`, `BuildBranch`, and `BuildTime` through Go ldflags.
- The release package metadata and binary metadata were not validated against `git rev-parse HEAD` during build.

## Fix

v1.0.6 changes release construction to a deterministic one-way provenance model:

- `scripts/build_release.sh` reads the current git commit, branch, and build time during release build.
- The package-local `config/version` is generated inside the release package.
- `go build` injects `x-ui/config.BuildCommit`, `BuildBranch`, and `BuildTime` via `-ldflags`.
- Build fails if package `config/version` or binary `x-ui version` does not match the source commit.

## Verification

Release verification must confirm:

- tag commit equals release source commit
- package `config/version` commit equals tag commit
- binary `x-ui version` Commit equals tag commit
- remote installed `x-ui version` Commit equals tag commit
