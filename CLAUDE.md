# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Additional Instructions — REQUIRED READING

**You MUST read every file in the `.claude/` folder whose name matches the pattern `CLAUDE-OD-*.md`.**
Do this at session start, before any other work.
Use `ls .claude/CLAUDE-OD-*.md` (or equivalent) to discover all matching files, then read each one.

These files extend and, where they conflict with the root `CLAUDE.md`, **override** its instructions.
Treat them as higher-priority guidance: if a rule here and a rule in `CLAUDE-OD-*.md` contradict each other, follow the `CLAUDE-OD-*.md` rule.

## The `checkmk_all` build tag

Every Go command in this repo needs `-tags checkmk_all`. Without it the build fails outright:

```txt
build constraints exclude all Go files in .../checkmk-api-spec@v0.0.5/generated/go/v2_2_0/p1
```

The tag is not used by this repo's own code (only `internal/client/version_types_test.go` carries it). It gates the companion module `github.com/BlackMesaLTD/checkmk-api-spec`, whose per-baseline packages are each guarded by `//go:build checkmk_all || checkmk_v2_4_0`-style constraints. The registry file in that module imports all baseline packages unconditionally, so with no tag they compile to empty and the imports dangle. `checkmk_all` compiles every CheckMK version's types in; the per-version tags exist to build a slimmer binary.

## Commands

```shell
go build -tags checkmk_all -o bin/terraform-provider-checkmk .   # Build binary (unversioned)
go build -tags checkmk_all -v ./...                              # Compile-check everything
go vet -tags checkmk_all ./...
golangci-lint run                                                # Reads build-tags from .golangci.yml
go test -tags checkmk_all -race -v -skip 'TestAcc' ./...         # Unit tests only
GOFLAGS='-tags=checkmk_all' go generate -tags checkmk_all ./...  # Formats examples/, regenerates docs/
```

Run `go generate` after any change to schemas, resource docs, or examples — CI fails if `docs/` is stale. The `GOFLAGS` prefix is required, not optional: `tfplugindocs` shells out to its own `go build` to export the provider schema, and that inner build ignores the outer `-tags` flag and fails on the excluded baseline packages.

Acceptance tests need `TF_ACC=1` plus `CHECKMK_URL`, `CHECKMK_USERNAME`, `CHECKMK_PASSWORD`. `scripts/run_tests.sh` derives them from local CheckMK Docker containers and starts the containers if needed:

```shell
./scripts/run_tests.sh --version 2.4 --verbose      # One version
./scripts/run_tests.sh --all                        # 2.2, 2.3 and 2.4
```

This is the Open Desk fork, so it carries its own version (`main.go`'s `version` default, currently `0.0.6-od.1`, overridable with `-ldflags "-X main.version=..."`). That value reaches `resp.Version` and the CheckMK `User-Agent`. Release builds go out under a versioned filename — see `README-OD-DEV.md` for the scheme and the current build's provenance.

Run a single test with `-run`:

```shell
go test -tags checkmk_all -run 'TestAccHostResource_FolderMove$' -v ./internal/provider
go test -tags checkmk_all -run 'TestMoveHost' -v ./internal/client   # Unit test, no CheckMK needed
```

Acceptance tests are named `TestAcc...` and live in `internal/provider`; unit tests are plain `Test...` and sit beside the code they cover.

## Architecture

A terraform-plugin-framework provider. `main.go` serves `provider.New(version)` and holds the `go:generate` hooks for docs.

### Layers

`internal/client/` wraps the CheckMK REST API (`/check_mk/api/1.0`). `internal/common/` is the shared contract every resource uses. The domain packages — `configuration/`, `hosts/`, `groups/`, `users/`, `labels/`, `rules/` (and `rules/wrappers/`) — own the resources. `internal/provider/` is glue plus two special resources (`checkmk_activation`, the `to_python` function).

### Version-aware types, and hollow mode

`client.NewClientWithOptions` calls `GetVersion` during construction and fails if the CheckMK version can't be detected, so `Client.Version` is always populated. `client.NewVersionedTypes(version)` maps that version onto a *baseline* package (e.g. `v2_4_0_p17`) of generated OpenAPI types — a baseline is a patch level where the schema changed enough to need its own types, and any version rounds to the nearest known one.

The provider's `type_mode` attribute (`internal/provider/provider.go`, resolved in `Configure`) decides what happens with those types:

| Mode | Behavior |
| --- | --- |
| `auto` (default) | Build types; on an unsupported version, warn and silently fall back to hollow |
| `static` | Build types; on an unsupported version, hard error |
| `hollow` | `Types = nil`, no plan-time validation, no tag promotion — the API is the sole arbiter |

The resolved mode and types land on `common.ProviderData`, which `Configure` assigns to *both* `resp.ResourceData` and `resp.DataSourceData`. `ProviderData` is the single object every resource receives.

Validation policy lives in `internal/common/validation.go`, not the client. `AttributeValidator.ShouldValidate()` gates on hollow mode. Unknown schema or unknown field means **allow** — custom attributes pass straight through to the API. Invalid enum values are errors; read-only and deprecated fields are warnings. Resources invoke this from `ValidateConfig`, guarded by `if r.providerData == nil { return }` since provider data can be absent during early validation.

### Resource registration

Registration is central aggregation, not a global registry. Each domain package exports `Resources()` / `DataSources()` in its `registration.go`, and `provider.go`'s `Resources()` appends them in turn. `rules/wrappers` is registered separately via `wrappers.AllResources()` because it imports `rules` for condition utilities and would otherwise form an import cycle — the same cycle is why `wrappers` re-extracts provider data by reflection (`wrappers/base.go`) instead of importing the provider package.

To add a resource: write `resource_<name>.go` in the right domain package, add its `New<Name>Resource` constructor to that package's `registration.go`, and — if it needs version-aware validation or doc generation — add a `SchemaInfo` entry to `common.SchemaRegistry` mapping the `checkmk_*` type name to its OpenAPI schema names. Touching `provider.go` is only necessary for a brand-new domain package.

### Activation

Mutations only *stage* changes in CheckMK; activation is a separate API call. Every mutating CRUD operation ends with `common.TrackAndActivate(...)`, which always increments a per-process `PendingChanges` counter and additionally calls `client.ActivateChanges` only when the effective `activate` is `"auto"`. In `"manual"` mode (the default) changes just accumulate until a `checkmk_activation` resource fires — it reads the pending changes, activates once, and clears the counter. Activation failure is a warning for ordinary resources but an error for `checkmk_activation`.

`activate`, `force_foreign_changes`, and `strict_resource_locking` are provider-level settings that individual resources may override; use `common.BuildBaseConfig` to resolve the effective value rather than reading `ProviderData` directly.

### ETags, drift, and the two request paths

`BuildETagHeaders(etag)` encodes the whole locking policy: a non-empty ETag becomes `If-Match: <etag>`, an empty one becomes `If-Match: *` (bypass). `common.FetchETagIfStrict` only fetches an ETag when strict locking is on.

Be aware the client has **two** HTTP paths. `request()` in `client.go` has the retry loop (429/500/502/503/504, exponential backoff with jitter) and sets `User-Agent`, but takes no custom headers, so it cannot send `If-Match` — it serves GETs, creates, and version detection. `requestWithHeaders()` in `activation.go` accepts headers and therefore handles every ETag-bearing mutation, but has no retry logic and sets no User-Agent. Writes are not retried.

Error mapping on writes must run in order: `HandlePreconditionFailed` (412 → `DriftError`) before `HandleNotFound` (404), then `handleResponse` maps the rest to `APIError`. DELETE treats 404 as idempotent success. Note the trap: `IsDriftError` tests for HTTP 428 on an `APIError`, while `DriftError` values come from 412 — two distinct drift signals coexist. On Read, `common.HandleNotFoundOnRead` turns a 404 into `RemoveResource`.

### CRUD: generic vs hand-written

`client/generic.go` offers Go-generic CRUD (`CreateResource`, `GetResourceWithETag`, `UpdateResource`, `DeleteResource`, `DeleteResourceAction`, `ListResources`) parameterized by a `ResourceConfig{CollectionPath, ObjectPathFmt, TypeName}`; roughly a dozen predefined configs sit at the bottom of that file. Prefer these for new simple resources. `host.go` and `folder.go` still carry hand-written CRUD because they need path normalization and move semantics the generic layer doesn't model.

Some resources delete via `POST .../actions/delete/invoke` rather than HTTP DELETE — that's what `DeleteResourceAction` is for.

### Folder paths and moves

Folder paths exist in two forms: user form `/A/B` and API form `~A~B`. `PathToAPIFormat` / `PathFromAPIFormat` (`client/folder.go`) convert between them, and `PathToAPIFormat` also normalizes `\` and `~` delimiters and maps root to `~`. Any value that reaches the API as a folder path — including a host's `target_folder` — must go through it.

Moving is an action endpoint, not a `PUT`. `MoveHost` posts to `/objects/host_config/{name}/actions/move/invoke` with a `target_folder` body field; `MoveFolder` posts to `/objects/folder_config/{apiPath}/actions/move/invoke` with a `destination` field. The field names genuinely differ between the two. Resources decide to move by diffing desired against current folder (`hosts/resource_host.go`, `configuration/resource_folder.go`) rather than forcing a replace. `client/move_test.go` pins the exact invoke paths and `~`-encoded bodies; test any new action endpoint the same way.

### Rule wrappers

Typed resources like `checkmk_host_check_interval` are sugar over the generic `checkmk_rule` — same `client.CreateRule`/`UpdateRule`/`DeleteRule` calls underneath. All CRUD, `ImportState`, and `ValidateConfig` logic lives once in `BaseRuleWrapper` (`wrappers/base.go`). A concrete wrapper embeds it and supplies a `RuleWrapperConfig`: the `Ruleset` string, a typed `ValueSchema`, and `ToValueRaw`/`FromValueRaw` converters marshalling the typed value to and from the generic rule's `value_raw`. `wrappers/conditions.go` re-exports the condition helpers from `rules` so conditions aren't duplicated. `groups/base.go` and `labels/base.go` follow the same "generic base plus a per-type config value" idiom; labels are themselves rule wrappers that serialize a label map into `value_raw`.

For the rule family, `id` is a computed deterministic hash of `(ruleset, description, conditions)` from `client.GenerateRuleHash`, while `api_id` is CheckMK's internal UUID used for update and delete. `id` deliberately omits `UseStateForUnknown` because it changes when description or conditions change; `api_id` uses it. Import sets `api_id` from the import ID and `id` to `"imported"`, and the next Read regenerates the real hash.

### Attribute promotion

Built-in host tag groups may be written unprefixed (`agent`) or prefixed (`tag_agent`). `common.AttributePromoter` promotes the unprefixed form to the API form on write and reverses it on read, so state keeps the key exactly as the user wrote it. It is driven by the generated schema field set and becomes a no-op in hollow mode. Custom tag groups must be written with an explicit `tag_` prefix.

## Conventions

Keep version-specific API behavior in `internal/client` or `internal/common` rather than scattering version checks across resources; `common.Features` (built from `version.AtLeast(...)`) exists for capability gating.

Resource files are `resource_<name>.go` / `data_source_<name>.go` with matching `*_test.go`. Declare the framework interface assertions at the top of the file (`var _ resource.Resource = ...`). Use the shared schema builders in `common/schema.go` (`ComputedIDAttribute`, `RequiredIDAttribute`, `ActivateAttribute`, …) rather than hand-rolling the plan modifiers, and the `common.AddClientError` / `AddValidationError` / `AddActivationWarning` helpers for diagnostics.

When reviewing a change, check that new logic preserves the previous semantics unless the behavior change is intentional and documented.

<!-- BEGIN .temp-kansio: hallinnoi open-desk-infra/workstations/WSL2/playbooks/05-repo-temp-symlinks.yml -->

## `.temp`-kansio

- `.temp` on repon paikallinen työkansio (esim. asennusten työtiedostot, dumpit ja viennit). Se ei ole gitissä (`.gitignore`), joten sen sisältöä ei palauteta kloonauksella
- JUKKAASUS-työaseman WSL:ssä `.temp` on symlinkki datalevylle: `.temp` → `/mnt/wsl/wsl-data/.temp/root/projects/opendesk/terraform-provider-checkmk`
- Fyysisesti tiedostot ovat datalevyllä `/mnt/wsl/wsl-data` (Windowsissa `C:\WSL-DATA\wsl-data.vhdx`), eivät WSL:n päälevyllä. Datalevy varmuuskopioidaan harvemmin kuin päälevy
- Jos `.temp` näkyy tyhjänä tai rikkinäisenä linkkinä, datalevyä ei ole liitetty (esim. `wsl --shutdown`in jälkeen). Liitä se: `cd /root/projects/opendesk/open-desk-infra/workstations/WSL2/playbooks && ansible-playbook -i inventory.ini 02-mount-wsl-disks.yml`
- Älä korvaa symlinkkiä tavallisella kansiolla. Jos `.temp` puuttuu, luo se samalla mallilla: kansio datalevylle, symlinkki, `.temp` `.gitignore`:en ja tämä kuvaus tähän tiedostoon. Kaikille repoille kerralla: `ansible-playbook -i inventory.ini 05-repo-temp-symlinks.yml -e dry_run=false`
- Lisätiedot: `open-desk-infra/workstations/WSL2/README-WSL2-Levytila.md`, osio "Lisälevyjen toteutussuunnitelma 15.11.2026"

<!-- END .temp-kansio: hallinnoi open-desk-infra/workstations/WSL2/playbooks/05-repo-temp-symlinks.yml -->
