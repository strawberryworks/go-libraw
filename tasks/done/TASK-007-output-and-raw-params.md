# TASK-007: Wrap Output And Raw Unpack Parameters

## Goal

Provide complete Go accessors and setters for `libraw_output_params_t` and `libraw_raw_unpack_params_t`.

## Context

Upstream exposes both direct parameter structs and setter helpers such as `libraw_set_demosaic`, `libraw_set_output_color`, `libraw_set_adjust_maximum_thr`, `libraw_set_user_mul`, `libraw_set_output_bps`, `libraw_set_gamma`, `libraw_set_no_auto_bright`, `libraw_set_bright`, `libraw_set_highlight`, `libraw_set_fbdd_noiserd`, and `libraw_set_output_tif`.

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
  - unrelated metadata structs.
- Output artifacts:
  - none.
- Default branch: `main`.
- Target branch: `feature/TASK-007-output-and-raw-params`.
- Package manager: Go modules.
- Runtime: Go with cgo enabled.
- Required environment: LibRaw.

## Scope

- Add Go structs/options mirroring all public fields of output and raw unpack params.
- Add safe setters/getters that keep Go API idiomatic.
- Add validation for array indexes and enum-like values where practical.
- Add inventory coverage for every field in both upstream structs.

## Out Of Scope

- Pixel processing algorithm changes.
- UI-level presets beyond small examples.

## Acceptance Criteria

- Given a user sets output color, bit depth, gamma, brightness, and demosaic algorithm, when processing runs, then the settings are applied through LibRaw.
- Given every field in `libraw_output_params_t` and `libraw_raw_unpack_params_t`, when inventory coverage is checked, then each field is wrapped or explicitly documented as unsupported.
- Given an invalid parameter index, when a setter is called, then Go returns or panics according to documented API policy consistently.

## Test Requirements

- Unit tests: required for option application and validation.
- Integration tests: required for at least one processing setting that changes LibRaw state.
- E2E tests: not required.
- Formatting: `gofmt` must be applied to changed Go files when Go files are changed.
- Lint: use `make lint` when available; otherwise use `go vet ./...`.
- Static analysis: `go vet ./...` is required for Go changes.
- Coverage scope: parameter wrapper code.
- Coverage metric: line coverage.
- Coverage target: at least `85%`.
- Required commands:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go test ./...`
  - `make check-api-inventory`

## Language Requirements

- Language: Go for implementation and tooling; Markdown for workflowr docs and task files.
- Style: idiomatic Go, simple package boundaries, explicit errors, table-driven tests when helpful.
- Dependency policy: prefer the Go standard library; do not add external dependencies unless explicitly required by the task.
- Standard tools: `gofmt`, `go vet`, `go test`, and project Makefile targets.

## Implementation Notes

Prefer option functions for construction-time settings and explicit methods for mutable settings after open.

## Clarifications

- Question: Should invalid indexes return errors or panic?
- Recommended default: return errors for user-provided runtime values; panic only for programmer-only fixed internal mistakes.
- Answer: return errors for public setters that accept runtime values.

## Implementation Outcome

- Added public `OutputParams` and `RawUnpackParams` mirrors with full get/set APIs.
- Added construction-time options and ergonomic mutable setters for LibRaw parameter helpers.
- Added validation for indexed setters, output bit depth, negative enum-like values, and fixed `p4shot_order` length.
- Documented field-level coverage, including unsupported `custom_camera_strings`.
- Verified parameter application through the processing pipeline.

## Git And PR

- Branch: `feature/TASK-007-output-and-raw-params`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels:
  - api
  - parameters
  - agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: direct struct mirrors can become unidiomatic.
- Mitigation: keep a low-level complete mirror and layer ergonomic helpers separately.
