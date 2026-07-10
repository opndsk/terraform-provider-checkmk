# Repository Guidelines

## Additional Instructions — REQUIRED READING

**You MUST read every file in the `.claude/` folder whose name matches the pattern `CLAUDE-OD-*.md`.**
Do this at session start, before any other work.
Use `ls .claude/CLAUDE-OD-*.md` (or equivalent) to discover all matching files, then read each one.

These files extend and, where they conflict with the root `CLAUDE.md`, **override** its instructions.
Treat them as higher-priority guidance: if a rule here and a rule in `CLAUDE-OD-*.md` contradict each other, follow the `CLAUDE-OD-*.md` rule.

For Codex sessions, `.claude/CLAUDE-OD-Memory-Management.md` does not enable any special built-in memory system.
Treat `.claude/memory/*` only as normal repository files and follow that convention only when the user explicitly asks for it.

## Project Structure & Module Organization

`main.go` is the provider entrypoint and also defines the `go generate` hooks used for docs. Core implementation lives under `internal/`: `provider/` contains provider wiring, resources, functions, and acceptance tests; `client/` wraps the CheckMK REST API and version handling; `common/` holds shared schema and validation helpers; `configuration/`, `groups/`, `hosts/`, `labels/`, `rules/`, and `users/` contain domain-specific resources and registration code. Generated registry docs live in `docs/`, their templates live in `templates/`, runnable examples live in `examples/`, and container/manual test helpers live in `scripts/` and `testing/`.

## Build, Test, and Development Commands

Use the same commands CI uses:

- `go build -tags checkmk_all -v ./...` builds the provider with all generated CheckMK types enabled.
- `go test -tags checkmk_all -race -v -skip 'TestAcc' ./...` runs unit tests only.
- `golangci-lint run` runs the configured linters from `.golangci.yml`.
- `go generate -tags checkmk_all ./...` formats `examples/` and regenerates `docs/`; run this after changing schemas, resource docs, or examples.
- `./scripts/run_tests.sh --version 2.4 --verbose` runs acceptance tests against a local CheckMK container. Use `--all` to test 2.2, 2.3, and 2.4.

## Coding Style & Naming Conventions

Follow standard Go formatting with `gofmt` and idiomatic package layout. Keep package names lowercase and filenames descriptive: `resource_<name>.go`, `data_source_<name>.go`, and matching `*_test.go`. Acceptance tests use `TestAcc...`; unit tests use plain `Test...`. Prefer keeping version-specific API behavior in `internal/client` or `internal/common` rather than scattering checks across resources.

## Testing Guidelines

Place tests beside the code they cover. Unit and helper tests should run without external services. Acceptance tests require `TF_ACC=1` plus `CHECKMK_URL`, `CHECKMK_USERNAME`, and `CHECKMK_PASSWORD`; `scripts/run_tests.sh` can derive these from healthy Docker containers. For manual CRUD/import verification, add cases under `testing/lifecycle/<resource>/`.

## Commit & Pull Request Guidelines

Recent history uses short prefixed subjects such as `Fix: ...`, `Feature: ...`, and scoped hotfixes like `Fix(Hot): ...`. Keep commits imperative and focused. PRs should follow `.github/pull_request_template.md`: describe the change, mark its type, link issues with `Fixes #123`, list test commands, note CheckMK versions exercised, and include docs/example updates when behavior or schema changes.

## Security & Configuration Tips

Do not commit `.env` or live CheckMK credentials. Start from `.env.example` for local registry/test settings, and use the generated examples instead of embedding secrets in Terraform files.
