# TASK-016: Build Fixture And Regression Test Suite

## Goal

Create a reliable test suite that validates wrapper behavior against real RAW fixtures and guards against regressions.

## Context

The repository already includes Canon, Nikon, Ricoh, Sony, and DNG fixtures. These should exercise opening, metadata, thumbnails, processing, and memory image paths.

## Execution Context

- Workspace: current workspace.
- Repository: current repository if one exists.
- Project root: current workspace root.
- Create new project if missing: no.
- Allowed paths:
  - `*.go`
  - `internal/**`
  - `testdata/**`
  - `docs/**`
  - `Makefile`
- Read-only paths:
  - original RAW fixture bytes unless adding approved new fixtures.
- Forbidden paths:
  - public API redesign.
- Output artifacts:
  - temporary outputs under `tmp/test-output/`.
- Default branch: `main`.
- Target branch: `feature/TASK-016-fixture-and-regression-suite`.
- Package manager: Go modules.
- Runtime: Go with cgo enabled.
- Required environment: LibRaw and bundled fixtures.

## Scope

- Add table-driven fixture tests for open, identify, metadata, thumbnail, unpack, and processing smoke paths.
- Add race-safe lifecycle tests.
- Add optional fuzz tests for invalid inputs around Go-level APIs.
- Add Makefile targets for fast tests and full fixture tests.

## Out Of Scope

- Large RAW fixture downloads.
- Exact color science golden image validation.

## Acceptance Criteria

- Given bundled fixtures, when full fixture tests run, then each fixture is opened and at least identified successfully.
- Given processing-capable fixtures, when processing smoke tests run, then outputs have valid dimensions and non-empty data.
- Given invalid inputs are fuzzed or table-tested, when errors occur, then they are typed and do not crash the process.

## Test Requirements

- Unit tests: required for invalid inputs and lifecycle.
- Integration tests: required for all bundled fixtures.
- E2E tests: optional example run if already available.
- Formatting: `gofmt` must be applied to changed Go files when Go files are changed.
- Lint: use `make lint` when available; otherwise use `go vet ./...`.
- Static analysis: `go vet ./...` is required for Go changes.
- Coverage scope: full module.
- Coverage metric: line coverage.
- Coverage target: report total and maintain at least the thresholds set by earlier tasks.
- Required commands:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`
  - `make test-fixtures`

## Language Requirements

- Language: Go for implementation and tooling; Markdown for workflowr docs and task files.
- Style: idiomatic Go, simple package boundaries, explicit errors, table-driven tests when helpful.
- Dependency policy: prefer the Go standard library; do not add external dependencies unless explicitly required by the task.
- Standard tools: `gofmt`, `go vet`, `go test`, and project Makefile targets.

## Implementation Notes

Separate fast tests from expensive fixture tests only if runtime becomes painful. Prefer `testing.Short()` gates over custom environment variables.

## Clarifications

- Question: Should more camera fixtures be added to cover every maker-note vendor?
- Recommended default: not in this task; use current fixtures and document gaps.
- Answer: do not add new camera fixtures in this task. Use the current bundled Canon, Nikon, Ricoh, Sony, and DNG fixtures and document the fixture policy.

## Implementation Outcome

- Added table-driven fixture regression tests for opening, identification, decoder info, processing, thumbnails, and invalid inputs.
- Added `make test-fast` and `make test-fixtures` targets.
- Documented fixture regression coverage and fixture policy in `docs/regression-tests.md`.

## Git And PR

- Branch: `feature/TASK-016-fixture-and-regression-suite`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels:
  - tests
  - fixtures
  - agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: RAW fixtures can make tests slow.
- Mitigation: profile runtime and split fast/full test targets if needed.
