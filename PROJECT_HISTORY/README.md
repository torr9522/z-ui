# Project History System

This directory is the permanent development audit record for this project.

It is not a replacement for Git. Git records code changes. Project History records the engineering context around important development phases:

- why the change was made
- what changed
- which files were added, modified, or removed
- database schema changes
- API changes
- UI changes
- installer and command changes
- systemd and nftables changes
- test results
- final commits
- file hashes for audit and recovery

## Purpose

Use this history for:

- secondary development
- rollback planning
- code audit
- project handoff
- AI-assisted continuation after context loss

## Workflow

Every major feature or development phase must add a new numbered file:

- `006_PORT_GUARD.md`
- `007_<FEATURE>.md`
- `008_<FEATURE>.md`
- `009_<FEATURE>.md`

Do not overwrite old phase records.

Do not rewrite old phase files after the phase is closed, except to correct obvious formatting mistakes. Future updates must be recorded in a new numbered phase file.

For every new phase, update:

- `PROJECT_HISTORY/LATEST_STATE.md`
- `PROJECT_HISTORY/MANIFEST_SHA256.txt`

The manifest should include all files related to the audited phase.
