# AI Agent Instructions

This repository contains a Go TUI for EOS, inspired by k9s.

## Project Rules

- Prefer the EOS CLI with machine-readable output for now.
- Do not reintroduce runtime gRPC dependencies unless explicitly requested.
- The app is primarily used on EOS hosts as `root`, but it should also support remote execution paths later.
- After code changes, run local checks and deploy the updated binary.

## Verification

- **CRITICAL**: Always run `go build ./...` and `go test ./...` after any code change and before declaring a task complete.
- Run `go test ./...`
- If relevant, run a local build with `go build ./...`
- Deploy with `make deploy-both`

## Deployment

- Default deploy target is `lobis-eos-dev`
- Secondary deploy target is `eospilot`
- Remote binary path is `/root/eos-tui`

## UI Guidance

- Keep EOS-facing column names aligned with the EOS CLI output.
- Prefer reusable Bubble Tea / Bubbles components where they fit naturally.
- Optimize for fast perceived performance: render views incrementally as data arrives.

## Style Rules (enforced by tests)

- `m.styles.header` (bold green, color 86) is **only** for the application title bar
  ("EOS TUI"). Never use it for column header rows inside views.
- All column header rows must go through `m.renderSimpleHeaderRow(columns, labels)`
  or `m.renderSelectableHeaderRow(...)`. Adding a `m.styles.header.Render(...)` call
  inside `renderBody` or any view-render function is a bug.
- The test `TestColumnHeadersUseConsistentStyle` enforces this rule across all views;
  add new views to the checklist in that test when adding new tabs.
