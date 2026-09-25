# Repository Guidelines

## Project Purpose & Architecture

Read `docs/design.md` before specification or architecture changes. Distinguish agreed behavior, provisional defaults, and unverified assumptions. The design is not implemented. See `docs/codex.md` for Codex setup.

`imgstamp` will stamp TIFF (including multipage), PNG, and JPEG files directly inside a directory. The initial deliverable is a CLI for Windows x64 and macOS ARM64, producing one PDF per input file.

Separate configuration, paper matching, compositing, PDF output, and batch orchestration from CLI parsing. Match pixel dimensions against configurable rules, not DPI. Never overwrite source files or existing outputs; skip existing PDFs. If any page fails, publish no PDF for that input and continue with the next file.

## Project Structure & Module Organization

The module is `imgstamp`, with Go `1.27.1` declared in `go.mod`. Documentation lives in `docs/`; `examples/stamp.toml` contains provisional configuration. No application source, tests, image assets, or CI configuration exist yet.

Use `cmd/imgstamp/` for the executable entry point and `internal/` for implementation packages as needed. Keep tests beside their source and small image fixtures in package-local `testdata/` directories.

## Build, Test, and Development Commands

Use a Go toolchain compatible with the version declared in `go.mod`. Once source packages exist, run these commands from the repository root:

- `go build ./...` — compile all packages.
- `go test ./...` — run all package tests.
- `go test -cover ./...` — inspect test coverage.
- `go vet ./...` — check for common correctness issues.
- `gofmt -w <file.go>` — format changed Go files.

No executable exists yet, so there is currently no local application run command.

## Coding Style & Naming Conventions

Use `gofmt`, lowercase package names, and descriptive identifiers. Document exported declarations and include file/page context in errors. Prefer the standard library; run `go mod tidy` after dependency changes. Use `path/filepath`. Avoid runtime dependencies on external image converters or Python.

## Testing Guidelines

Use Go's `testing` package, `*_test.go`, and `TestXxx`. Cover paper-matching tolerances and ambiguity, multipage TIFF, stamp readability, PDF dimensions, existing-output skips, and file-level failure cleanup. Use deterministic fixtures and `t.TempDir()`. Render PDFs for visual checks. No coverage threshold is configured. Real drawing compatibility and printed output remain unverified until representative samples are available.

## Commit & Pull Request Guidelines

Keep `main` as the baseline and develop changes on focused `feat/`, `fix/`, or `docs/` branches. Use concise, imperative commit subjects, such as `Add PNG decoding tests`. Keep changes focused.

Pull requests should explain the change, its motivation, and validation performed. Link relevant issues when available. Include before-and-after images when changing visible image output, and disclose any checks that could not be run.
