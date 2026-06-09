# TASK-010: Wrap Image, Raw, And Thumbnail Memory Access

## Goal

Expose safe Go access to processed images, raw image buffers, thumbnail buffers, and color filter helpers.

## Context

LibRaw provides processed image memory via `libraw_processed_image_t`, raw image data through `libraw_rawdata_t`, thumbnail data through `libraw_thumbnail_t` and thumbnail lists, plus helpers such as `libraw_COLOR`.

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
  - unrelated build tooling.
- Output artifacts:
  - temporary decoded images under `tmp/outputs/`.
- Default branch: `main`.
- Target branch: `feature/TASK-010-image-raw-thumbnail-memory-access`.
- Package manager: Go modules.
- Runtime: Go with cgo enabled.
- Required environment: LibRaw and bundled fixtures.

## Scope

- Add Go wrappers for processed image data including type, dimensions, colors, bits, data size, and data bytes.
- Add thumbnail accessors with safe byte copying or documented lifetime rules.
- Add raw image buffer accessors where LibRaw exposes data.
- Add color filter helper wrappers.

## Out Of Scope

- High-level image manipulation.
- Lossless conversion libraries beyond standard Go image support where applicable.

## Acceptance Criteria

- Given a processed memory image, when data is read into Go, then size and format metadata match LibRaw's struct.
- Given a thumbnail is unpacked, when thumbnail bytes are requested, then callers can persist them after the LibRaw handle is closed if the API claims copied ownership.
- Given `COLOR(row, col)` is called, when row/col are valid, then LibRaw's color index is returned.

## Test Requirements

- Unit tests: required for byte ownership and bounds checks.
- Integration tests: required with at least one bundled fixture.
- E2E tests: not required.
- Formatting: `gofmt` must be applied to changed Go files when Go files are changed.
- Lint: use `make lint` when available; otherwise use `go vet ./...`.
- Static analysis: `go vet ./...` is required for Go changes.
- Coverage scope: memory access wrappers.
- Coverage metric: line coverage.
- Coverage target: at least `80%`.
- Required commands:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`

## Language Requirements

- Language: Go for implementation and tooling; Markdown for scenarum docs and task files.
- Style: idiomatic Go, simple package boundaries, explicit errors, table-driven tests when helpful.
- Dependency policy: prefer the Go standard library; do not add external dependencies unless explicitly required by the task.
- Standard tools: `gofmt`, `go vet`, `go test`, and project Makefile targets.

## Implementation Notes

Do not expose unsafe slices into C memory unless the method name and documentation make the lifetime impossible to miss.

## Clarifications

- Question: Should APIs prefer zero-copy unsafe views or copied Go byte slices?
- Recommended default: copied slices by default; optional unsafe view methods may be added with explicit naming.
- Answer: Copied slices. `RawImage`, `ThumbnailData`, and `MemImage`/`MemThumb` all return Go-owned copies that remain valid after Close. No zero-copy unsafe views are exposed in this task; they can be added later behind explicit `Unsafe*` naming if needed.

## Implementation Outcome

- Added `Processor.Color` (libraw_COLOR), `RawWidth`/`RawHeight` (libraw_get_raw_*), `RawImage` (copied single-channel Bayer samples from rawdata.raw_image, row-padded length (raw_pitch/2)*raw_height), and `ThumbnailData` (copied thumbnail bytes from thumbnail.thumb).
- Processed-image bytes were already wrapped by `MemImage`/`MemThumb` (TASK-006); raw/thumbnail summaries by metadata (TASK-008). This task adds the actual byte/sample buffers, which the coverage map previously listed as "data pointer not exposed".
- All buffers are copied into Go memory; `ErrNoImageData` is returned when the relevant decode step has not run or the format lacks that buffer. Tests confirm copies survive Close.

## Git And PR

- Branch: `feature/TASK-010-image-raw-thumbnail-memory-access`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels:
  - image-data
  - cgo
  - agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: dangling pointer exposure can cause crashes.
- Mitigation: copy by default and test post-close access behavior.

