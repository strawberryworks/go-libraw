# go-libraw Scenarum Queue

Scenarum tool repository: https://github.com/ivanglie/scenarum

This queue defines the work needed to build `github.com/ivanglie/go-libraw` as a complete Go wrapper for upstream `LibRaw/LibRaw`.

The upstream inventory baseline for the initial task set is `LibRaw/LibRaw` HEAD observed on 2026-06-08, reporting `LIBRAW_VERSION_STR` as `0.22.0-Release` in `libraw/libraw_version.h`. Public headers considered in scope:

- `libraw/libraw.h`
- `libraw/libraw_const.h`
- `libraw/libraw_types.h`
- `libraw/libraw_version.h`
- `libraw/libraw_alloc.h`
- `libraw/libraw_datastream.h`

Use `AGENTS.md` and take tasks from `tasks/inbox/` one at a time.
