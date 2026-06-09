# TASK-003: Generate Upstream API Inventory

## Goal

Create a reproducible inventory of upstream LibRaw public API and a coverage gate that makes "100% wrapper" measurable.

## Context

Manual tracking will drift. The wrapper should parse or otherwise inventory `libraw/libraw.h`, `libraw/libraw_const.h`, `libraw/libraw_types.h`, and `libraw/libraw_version.h`, then compare the inventory against Go bindings.

## Execution Context

- Workspace: current workspace.
- Repository: current repository if one exists.
- Project root: current workspace root.
- Create new project if missing: no.
- Allowed paths:
  - `cmd/**`
  - `internal/**`
  - `docs/**`
  - `tools/**`
  - `testdata/**`
  - `Makefile`
  - `go.mod`
  - `go.sum`
- Read-only paths:
  - `tasks/**`
- Forbidden paths:
  - generated files outside documented generated paths.
- Output artifacts:
  - `docs/libraw-api-inventory.md`
  - `internal/generated/**` if generated bindings are introduced.
- Default branch: `main`.
- Target branch: `feature/TASK-003-upstream-api-inventory-generator`.
- Package manager: Go modules.
- Runtime: Go with access to LibRaw headers.
- Required environment: local LibRaw headers or checked-in fixture headers for tests.

## Scope

- Add a tool that inventories public C API functions, enums, macros, structs, and version symbols.
- Record the upstream version and header paths used for generation.
- Add a coverage report showing wrapped, intentionally internal, deferred, and unsupported symbols.
- Add a CI/check command that fails when the inventory changes without updating the coverage map.

## Out Of Scope

- Wrapping every symbol in this task.
- Parsing private implementation headers beyond what is needed to explain exclusions.

## Acceptance Criteria

- Given LibRaw headers are available, when the inventory command runs, then it writes a deterministic API inventory.
- Given a symbol is present upstream but not mapped, when the coverage gate runs, then the report identifies it.
- Given generated output is committed, when the command is rerun without upstream changes, then the diff is empty.

## Test Requirements

- Unit tests: required for parser or extractor logic using fixture header snippets.
- Integration tests: required for real LibRaw headers when available.
- E2E tests: not required.
- Formatting: `gofmt` must be applied to changed Go files when Go files are changed.
- Lint: use `make lint` when available; otherwise use `go vet ./...`.
- Static analysis: `go vet ./...` is required for Go changes.
- Coverage scope: inventory tool and coverage gate logic.
- Coverage metric: line coverage.
- Coverage target: at least `80%`.
- Required commands:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go test ./...`
  - `make check-api-inventory`

## Language Requirements

- Language: Go for implementation and tooling; Markdown for scenarum docs and task files.
- Style: idiomatic Go, simple package boundaries, explicit errors, table-driven tests when helpful.
- Dependency policy: prefer the Go standard library; do not add external dependencies unless explicitly required by the task.
- Standard tools: `gofmt`, `go vet`, `go test`, and project Makefile targets.

## Implementation Notes

Prefer structured parsing via clang tooling if practical; otherwise keep the extractor conservative and heavily fixture-tested. The coverage map should be human-readable.

## Clarifications

- Question: Should the project vendor a snapshot of LibRaw headers for reproducible inventory tests?
- Recommended default: yes, vendor minimal header fixtures under `testdata/headers/` rather than the full LibRaw source.
- Answer:

## Git And PR

- Branch: `feature/TASK-003-upstream-api-inventory-generator`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels:
  - tooling
  - api-coverage
  - agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: C/C++ header parsing is complex.
- Mitigation: start with public C API and exact fixture coverage, then expand parser capability only as needed.

