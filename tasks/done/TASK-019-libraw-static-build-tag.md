# TASK-019: `libraw_static` build tag for static linking

## Goal

Provide an opt-in `libraw_static` build tag that links LibRaw and its open-source
dependencies statically, so downstream apps can ship a self-contained binary and
end users do not have to install LibRaw. Default builds stay dynamic and unchanged.

## Context

Today the cgo bridge links LibRaw dynamically only. `internal/librawc/librawc.go`
carries the link directives in its preamble:

```
#cgo linux pkg-config: libraw
#cgo darwin,arm64 CFLAGS: -I/opt/homebrew/opt/libraw/include
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/libraw/lib -lraw
#cgo darwin,amd64 CFLAGS: -I/usr/local/opt/libraw/include
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/libraw/lib -lraw
#include <stdlib.h>
#include <libraw/libraw.h>
```

`-lraw` resolves to `libraw.dylib`, and these directives live inside this (internal)
package, so a downstream module cannot force static linking on its own. This task
adds that capability here, where the cgo link policy belongs.

Consumer driving this: the Strawberry photo editor (wired via `replace =>
../go-libraw`) wants a self-contained macOS `.app` built with `-tags libraw_static`.

Static dependency facts (macOS, from `otool -L libraw.dylib`): LibRaw pulls
`libomp` (OpenMP), `jpeg-turbo` (NOT plain `jpeg`), `little-cms2`, plus the system
`libz` / `libc++` / `libSystem`. Static archives are present under Homebrew:
`libraw.a`, `jpeg-turbo/libjpeg.a`, `little-cms2/liblcms2.a`, `libomp/libomp.a`.
System libs stay dynamic (present on every Mac; full static libc is not the macOS
convention).

## Execution Context

- Workspace: `code/`
- Repository: `code/go-libraw`
- Project root: `code/go-libraw`
- Create new project if missing: no
- Allowed paths:
  - `internal/librawc/**`
  - `Makefile`
  - `docs/**`
  - `*.go` (only if a build-tag smoke test is added at the package level)
- Read-only paths: RAW fixtures, generated inventories
- Forbidden paths: broad API/coverage-map changes
- Output artifacts: none committed (built test binaries go to `tmp/`)
- Default branch: `main`
- Target branch: `feature/TASK-019-libraw-static-build-tag`
- Package manager: Go modules
- Runtime: Go with cgo enabled
- Required environment: Homebrew LibRaw + `jpeg-turbo`, `little-cms2`, `libomp`
  (their `.a` archives) for the static smoke build

## Scope

- Split the cgo link directives out of `librawc.go` into two tag-gated, directive-only
  files (each is just a preamble + `import "C"`; `#cgo` CFLAGS/LDFLAGS are package-global
  so they still apply to `librawc.go`'s `#include`):
  - `librawc_cgo_dynamic.go` — `//go:build cgo && !libraw_static` — the current dynamic
    directives (linux `pkg-config: libraw`; darwin `-lraw`).
  - `librawc_cgo_static.go` — `//go:build cgo && libraw_static` — static directives.
- Leave the `#include <stdlib.h>` / `#include <libraw/libraw.h>` lines in `librawc.go`;
  move only the `#cgo` lines.
- Static darwin LDFLAGS (full `.a` paths force static selection):
  - arm64: `/opt/homebrew/opt/libraw/lib/libraw.a /opt/homebrew/opt/jpeg-turbo/lib/libjpeg.a /opt/homebrew/opt/little-cms2/lib/liblcms2.a /opt/homebrew/opt/libomp/lib/libomp.a -lz -lc++`
  - amd64: the same archives under `/usr/local/opt/...`
  - static CFLAGS: the existing `-I.../libraw/include` per arch.
- Allow override via the standard `CGO_CFLAGS`/`CGO_LDFLAGS` env so a caller's Makefile
  can resolve prefixes with `brew --prefix` instead of the hardcoded paths.
- Add a `Makefile` target (e.g. `libraw-check-static` / `build-static`) that builds
  with `-tags libraw_static` and runs the linked-version smoke test.
- Document the tag in `docs/libraw-build.md` (macOS static link line; note Linux
  static needs `PKG_CONFIG="pkg-config --static"` or explicit LDFLAGS; Windows TBD).

## Out Of Scope

- Linux/Windows static linking implementation (document the approach only; macOS first).
- Code signing / notarization.
- Changing default (dynamic) behavior or the public Go API.

## Acceptance Criteria

- Given `go build ./...` with no tags, when built, then linking is unchanged (dynamic).
- Given `go build -tags libraw_static ./...` on macOS arm64 with the `.a` deps present,
  when built, then it compiles and links.
- Given a tiny test binary built with `-tags libraw_static`, when inspected with
  `otool -L`, then it shows no `/opt/homebrew/...` or `/usr/local/...` dynamic entries
  (only `/usr/lib/*` and `/System/...`).
- Given Homebrew LibRaw is unlinked, when the static test binary runs, then the linked
  version smoke test still passes.

## Test Requirements

- Unit tests: none new (link-policy change; no Go logic added).
- Integration tests: `TestLinkedVersion` must pass under `-tags libraw_static`.
- E2E tests: a scripted static smoke build + `otool -L` assertion (Makefile target).
- Formatting: `gofmt` on changed Go files.
- Lint: `make lint` if available, else `go vet ./...` (both with and without the tag).
- Static analysis: `go vet ./...` and `go vet -tags libraw_static ./...`.
- Coverage scope: n/a (no new Go logic); state this explicitly.
- Coverage metric: n/a.
- Coverage target: n/a (exception: build-policy change carries no testable Go logic).
- Required commands:
  - `go build ./...`
  - `go build -tags libraw_static ./...`
  - `go test -tags libraw_static -run TestLinkedVersion ./internal/librawc`
  - `make build` (dynamic, unchanged) and the new static target

## Language Requirements

- Language: Go (cgo directives), Makefile, Markdown.
- Style: idiomatic Go; minimal, additive; preserve the dynamic default.
- Dependency policy: no new Go dependencies.
- Standard tools: `gofmt`, `go vet`, `go build`, `go test`, `make`, `otool`, `brew`.

## Implementation Notes

`#cgo CFLAGS`/`LDFLAGS` are collected package-wide across all files, so the
directive-only files apply to every C compilation/link unit in `librawc` regardless
of which file holds the `#include`. A directive-only file is valid with just the
preamble comment and `import "C"`. Verify both build modes still satisfy the `!cgo`
stub files (the `cgo` build constraint is unchanged; only `libraw_static` is added).

## Clarifications

- Question: Hardcode Homebrew prefixes in the static directives, or require the caller
  to pass `CGO_LDFLAGS`?
- Recommended default: hardcode the standard per-arch Homebrew prefixes (consistent
  with the existing dynamic directives) AND honor `CGO_*` env overrides for callers
  that resolve `brew --prefix` themselves.
- Answer: Approved by user (path "A", 2026-06-09): add this tag to go-libraw.

## Git And PR

- Branch: `feature/TASK-019-libraw-static-build-tag`
- Commit style: Conventional Commits
- PR target: `main`
- PR provider: GitHub
- PR tool: GitHub CLI (`gh`)
- PR mode: draft
- Push branch automatically: yes, after required checks pass
- PR labels: build, cgo, agent-generated
- PR reviewers:
- PR assignees:
- Review requirements: self-review before PR

## Risks

- Risk: static link surfaces unresolved transitive symbols (jpeg-turbo, lcms2, omp, c++).
- Mitigation: full `.a` paths + `-lz -lc++`; iterate on link errors; assert via `otool -L`.
- Risk: moving `#cgo` lines accidentally drops CFLAGS from `librawc.go`'s `#include`.
- Mitigation: rely on package-global flag collection; verify a clean `go build ./...`
  in both modes before delivery.
