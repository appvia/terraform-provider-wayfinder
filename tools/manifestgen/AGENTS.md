# tools/manifestgen

## OVERVIEW
Reflects Wayfinder API types at the pinned tag into `schema/wayfinder-manifest.json`. Compiled ONLY inside `.wayfinder-src/hack/tfprovider-manifestgen/` by `scripts/manifestgen.sh`; every `.go` file is `//go:build wfmanifestgen`.

## WHERE TO LOOK
| Task | File | Notes |
|------|------|-------|
| Classify kinds | kinds.yaml | typed / manifest / exclude; rules in DEVELOPMENT.md "Classifying kinds" |
| Per-field Terraform overrides | fields.yaml | immutable / computed / ignore(+reason); `path` starts `spec.` |
| CLI flags (`-kinds`, `-fields`, `-wayfinder-version`, `-wayfinder-commit`, `-o`, `-list-kinds`) | main.go | Extra args via `./scripts/manifestgen.sh run <args>` |
| Pipeline: validate → coverage → reflect → overrides | generate.go | `applyOverrides` fails on stale paths |
| Go type → `manifest.Field` | reflect.go | snake-casing mirrors `convert.SnakeCase` |
| Validation/encryption tags | metadata.go | `+encrypted` → `sensitive: true` |
| kinds.yaml ⇄ registry coverage | coverage.go | `CoverageError` lists missing/extra kinds |
| YAML schema + strict loading | overrides.go | Unknown keys, unsorted, duplicates all fail |
| Controller inventory | controllers.go | `waitForReady` = hasStatus && kind has a controller && !noWait |
| Vars scope discovery + coverage | varscopes.go | Parses registered `varsHandler` constructors in handler source; compares against `kinds.yaml` `varScopes` |

## CONVENTIONS
- Imports `github.com/appvia/wayfinder/...`, including `.../hack/tfprovider-manifestgen/manifest` — that is `internal/manifest/contract.go` copied in by the script. Contract changes must stay stdlib-only.
- `kinds.yaml` sorted by group then kind, `fields.yaml` by group, kind, path. Must match the registry exactly; a Wayfinder bump adding/removing a kind fails until classified.
- `kinds.yaml` `varScopes` sorted by scope, unique, `typed` or `exclude` (reason required). Must match registered vars handlers; only typed scopes enter the manifest.
- `reason` required for `exclude`, `noWait`, and `ignore`. `name`/`noWait` rejected on excluded kinds.
- Inspect output without touching `schema/`: `make manifest MANIFEST_OUT=/tmp/m.json`.

## ANTI-PATTERNS
- `go build`, `go test`, or `go mod tidy` here; the nested `go.mod` only fences it out of the root module. Use `make manifestgen-test`.
- Importing anything from this dir (or the private module) in the root module — `scripts/check-no-private-deps.sh` fails CI.
- Importing any Wayfinder handlers package for vars scope discovery; parse source with `go/parser` and `go/ast` instead.
- Using `fields.yaml` `ignore` to hide a secret that tfgen can't model; fix tfgen or classify the kind `manifest`.
