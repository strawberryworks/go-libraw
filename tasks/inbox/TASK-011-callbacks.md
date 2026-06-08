# TASK-011: Wrap Progress, Data Error, EXIF, And Maker Notes Callbacks

## Goal

Expose LibRaw callback hooks to Go safely.

## Context

Public C callbacks include `libraw_set_exifparser_handler`, `libraw_set_makernotes_handler`, `libraw_set_dataerror_handler`, and `libraw_set_progress_handler`.

## Execution Context

- Workspace: current workspace.
- Repository: current repository if one exists.
- Project root: current workspace root.
- Create new project if missing: no.
- Allowed paths:
  - `*.go`
  - `internal/**`
  - `docs/**`
  - `testdata/**`
- Read-only paths:
  - original RAW fixtures.
- Forbidden paths:
  - broad processing changes except callback-triggering test setup.
- Output artifacts:
  - none.
- Default branch: `main`.
- Target branch: `feature/TASK-011-callbacks`.
- Package manager: Go modules.
- Runtime: Go with cgo enabled.
- Required environment: LibRaw and fixtures.

## Scope

- Add Go callback registration APIs.
- Implement cgo callback trampolines with safe handle lookup and cleanup.
- Support progress callback cancellation semantics if LibRaw allows non-zero callback return to stop work.
- Add documentation for callback threading, panics, and lifetime.

## Out Of Scope

- Arbitrary C callback plugin APIs beyond LibRaw public callbacks.
- Async processing orchestration.

## Acceptance Criteria

- Given a progress callback is registered, when processing a fixture, then Go receives progress events.
- Given a callback returns cancellation, when LibRaw honors it, then Go receives the corresponding error.
- Given a handle is closed, when callbacks are released, then callback registries do not retain the handle.
- Given a Go callback panics, when invoked from C, then the wrapper recovers and returns a documented error path.

## Test Requirements

- Unit tests: required for callback registry lifecycle.
- Integration tests: required for progress callback during fixture processing.
- E2E tests: not required.
- Formatting: `gofmt` must be applied to changed Go files when Go files are changed.
- Lint: use `make lint` when available; otherwise use `go vet ./...`.
- Static analysis: `go vet ./...` is required for Go changes.
- Coverage scope: callback registry and public callback API.
- Coverage metric: line coverage.
- Coverage target: at least `80%`.
- Required commands:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`

## Language Requirements

- Language: Go for implementation and tooling; Markdown for workflowr docs and task files.
- Style: idiomatic Go, simple package boundaries, explicit errors, table-driven tests when helpful.
- Dependency policy: prefer the Go standard library; do not add external dependencies unless explicitly required by the task.
- Standard tools: `gofmt`, `go vet`, `go test`, and project Makefile targets.

## Implementation Notes

Use `runtime/cgo.Handle` or an equivalent safe registry. Avoid passing Go pointers to C in violation of cgo rules.

## Clarifications

- Question: Should callbacks be allowed to run concurrently?
- Recommended default: document that LibRaw may invoke callbacks from processing calls; make registry safe but avoid promising callback order beyond LibRaw behavior.
- Answer:

## Git And PR

- Branch: `feature/TASK-011-callbacks`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels:
  - callbacks
  - cgo
  - agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: cgo callback misuse can crash the process.
- Mitigation: keep trampolines tiny, recover panics, and race-test lifecycle cleanup.

