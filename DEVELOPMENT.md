# Development guide

This document is for Appvia engineers and contributors working on the Wayfinder Terraform provider.

## Manifest contract

The manifest is the JSON document the Wayfinder manifest generator produces from a pinned Wayfinder release. The provider reads it to decide which resources and data sources to generate. The Go types live in `internal/manifest/contract.go`. That file imports only the standard library, because the generator copies it verbatim into the Wayfinder checkout. Do not add non-stdlib imports to it. `TestContractImportsAreStdlib` enforces this.

The golden fixture `internal/manifest/testdata/golden-manifest.json` is the shared reference document. Tests use it to pin the wire format.

### Encoding rules

- JSON keys are camelCase. Optional keys are omitted when empty (`omitempty`).
- Exceptions: `Kind.typed`, `Kind.versioned`, `Kind.hasStatus`, and `Kind.waitForReady` are always written, so `false` is explicit. `Manifest.manifestVersion`, `wayfinderVersion`, and `kinds` are always written too.
- Output is 2-space indented with a trailing newline (`manifest.Encode`).
- `Field.properties` is emitted in document order. Decoding records the order in `Field.PropertyOrder`. Encoding emits `PropertyOrder` first, then any names not listed, sorted. `propertyOrder` is never written as a key.
- Decoding is strict: unknown keys are errors at every level.

### Top-level document

| Key | Go field | Type | Rules |
|---|---|---|---|
| `manifestVersion` | `ManifestVersion` | integer | Must be `1`. |
| `wayfinderVersion` | `WayfinderVersion` | string | Required. The pinned Wayfinder tag, for example `v3.3.2`. |
| `wayfinderCommit` | `WayfinderCommit` | string | Optional. If set, 40 lowercase hex characters: the peeled commit SHA. |
| `kinds` | `Kinds` | array of `Kind` | Sorted by `kind`, unique. |
| `excluded` | `Excluded` | array of string | Optional. Sorted, unique, and disjoint from `kinds`. These are kinds classified as excluded: the provider can report "cannot be managed" for them instead of "upgrade the provider". |
| `varScopes` | `VarScopes` | array of string | Optional. Sorted, unique, lowercase (matches `^[a-z][a-z0-9]*$`). The variable and secret scopes Wayfinder serves that the provider supports, classified in `tools/manifestgen/kinds.yaml`. Not restricted to a fixed set. |

### Kind

| Key | Go field | Type | Rules |
|---|---|---|---|
| `kind` | `Kind` | string | PascalCase, matches `^[A-Z][A-Za-z0-9]*$`. |
| `group` | `Group` | string | Required. API group. |
| `version` | `Version` | string | Required. API version, for example `v3`. |
| `apiPath` | `APIPath` | string | Required. Must start with `/`. |
| `typed` | `Typed` | boolean | `true`: a typed resource and data source are generated. `false`: the kind is manageable only through `wayfinder_manifest`. |
| `scopes` | `Scopes` | array of string | At least one. Each is one of `tenant`, `workspace`, `environment`. No duplicates. |
| `versioned` | `Versioned` | boolean | The kind carries `metadata.version` (strict semver). |
| `hasStatus` | `HasStatus` | boolean | The kind reports status. |
| `waitForReady` | `WaitForReady` | boolean | The provider waits for readiness. Requires `hasStatus`. |
| `terraformName` | `TerraformName` | string | Required when `typed` is `true`: lowercase snake case, matches `^[a-z][a-z0-9_]*$`. Must be empty when `typed` is `false`. |
| `description` | `Description` | string | Optional. Used for generated documentation. |
| `spec` | `Spec` | Field | Required when `typed` is `true`, and must have `type` `object`. Optional for manifest-only kinds. Describes the spec subtree only, never `metadata` or `status`. |

### Field

A `Field` is one node of a spec schema.

| Key | Go field | Type | Rules |
|---|---|---|---|
| `type` | `Type` | string | Required. One of `string`, `boolean`, `integer`, `number`, `object`, `array`, `map`, `json`. |
| `description` | `Description` | string | Optional. |
| `required` | `Required` | boolean | The field must be set by users. |
| `computed` | `Computed` | boolean | The server sets the value. |
| `immutable` | `Immutable` | boolean | Changing the value forces replacement. |
| `sensitive` | `Sensitive` | boolean | The value is secret. |
| `enum` | `Enum` | array of string | Allowed values. No duplicates. |
| `enumDocs` | `EnumDocs` | array of `{value, description}` | Documentation per enum value. Each `value` must appear in `enum`. |
| `pattern` | `Pattern` | string | Must compile as a Go regular expression. |
| `minLength`, `maxLength` | `MinLength`, `MaxLength` | integer | Each `>= 0`. Min must not exceed max. |
| `minimum`, `maximum` | `Minimum`, `Maximum` | number | Min must not exceed max. |
| `format` | `Format` | string | Optional format hint, for example `date-time` or `uri`. |
| `properties` | `Properties` | object of `Field` | Only when `type` is `object`. Keys match `^[A-Za-z_][A-Za-z0-9_]*$`. |
| `items` | `Items` | `Field` | Required when `type` is `array`, and only allowed then. |
| `values` | `Values` | `Field` | Required when `type` is `map`, and only allowed then. |

### Validation

`Manifest.Validate` returns the first violation. Its message starts with the JSON path of the offending value, for example:

```
kinds[2].spec.properties.foo.type: unknown type "x"
```

`manifest.Parse` decodes and validates in one step.

### Golden fixture

`internal/manifest/testdata/golden-manifest.json` covers the four kind shapes the provider must handle:

- `Workspace`: tenant-scoped, typed, with nested objects, an array of objects, enums with docs, and `computed` and `immutable` fields.
- `Integration`: workspace and environment scopes, typed, with a sensitive string, a sensitive `map`, and a `json` field.
- `Blueprint`: versioned and typed.
- `Pipeline`: manifest-only (`typed` is `false`).

`excluded` lists `Run` and `Session`.

`varScopes` lists the four variable and secret scopes: `environment`, `instance`, `tenant`, `workspace`.

<!-- Add further sections below this one. Keep the "Manifest contract" heading as is. -->

## Repository layout

| Path | What it is |
|---|---|
| `WAYFINDER_REF` | The pinned Wayfinder release tag, one line (for example `v3.3.2`). |
| `scripts/` | Checkout, generation, bump and guard scripts. Run them through `make` where a target exists. |
| `tools/manifestgen/` | The manifest generator plus `kinds.yaml` and `fields.yaml`. A nested Go module whose files all carry `//go:build wfmanifestgen`. It only compiles inside the Wayfinder checkout. |
| `schema/wayfinder-manifest.json` | The committed, generated manifest. Never edit it by hand. |
| `internal/manifest/` | The manifest contract (see above). |
| `cmd/tfgen/` | Turns the manifest into Go schema code and examples. Also prints schema diffs (`-diff`). |
| `internal/generated/` | Output of `go generate ./...`. Never edit it by hand. |
| `internal/fakeapi/` | In-process fake Wayfinder API used by unit tests. |
| `.wayfinder-src/` | Disposable, gitignored checkout of the pinned Wayfinder tag. Never commit anything from it. |

## How generation works

The provider's resources come from the Wayfinder release pinned in `WAYFINDER_REF`. The pipeline has four steps.

1. `scripts/wayfinder-checkout.sh` (`make wayfinder-src`) validates `WAYFINDER_REF`, resolves the tag to its peeled commit with `git ls-remote`, and makes a shallow checkout of exactly that tag in `.wayfinder-src/`. If the checkout is already at that commit it is reused.
2. `scripts/manifestgen.sh` (`make manifest`) copies `tools/manifestgen/*.go`, `tools/manifestgen/testdata/` and `internal/manifest/contract.go` into `.wayfinder-src/hack/tfprovider-manifestgen/`. It then runs `go run -tags wfmanifestgen ./hack/tfprovider-manifestgen` from inside `.wayfinder-src` with `GOTOOLCHAIN=auto`. Wayfinder's own `go.mod` and `go.sum` govern that build, so the provider module never depends on the private Wayfinder module. The generator reflects every registered kind, applies `kinds.yaml` and `fields.yaml`, and writes `schema/wayfinder-manifest.json` with `wayfinderVersion` (the tag) and `wayfinderCommit` (the peeled SHA).
3. `go generate ./...` (`make generate`) runs `cmd/tfgen`, which reads the manifest and writes `internal/generated/` and the examples.
4. `make docs` runs `tfplugindocs generate` and `tfplugindocs validate` (the terraform CLI, 1.11 or later, must be on `PATH`), then `scripts/check-docs.sh`, which fails if a resource or data source page has no example. Pages in `docs/` are generated from the provider schema, `templates/` and `examples/`; edit those, never `docs/`. CI fails when committed docs drift.

### Hand-written examples

`examples/provider/` and the `examples/` directories for `wayfinder_manifest`, `wayfinder_variable`, `wayfinder_secret` and `wayfinder_feature_integration` are written by hand. tfgen only deletes example files that start with its `Code generated by tfgen` header, so it leaves these alone.

### Wording rules

Upstream descriptions are written for everyone who reads the Wayfinder API reference, and a few phrases suit Wayfinder operators more than SaaS users. `cmd/tfgen/wording.go` holds an ordered list of regex rewrites applied to every generated description. After the rewrites, tfgen fails if a description still matches the forbidden-wording list (the internal variable-store kind name, "your administrator", "this instance", `WF_*` variables, Wayfinder Helm values). When a Wayfinder bump trips the check, add a narrow rule with a comment saying why, and a test.

Useful variants:

```sh
make manifest MANIFEST_OUT=/tmp/manifest.json   # write somewhere else, leave schema/ alone
./scripts/manifestgen.sh run -list-kinds        # extra args go to the generator; later flags win
make manifestgen-test                           # generator unit tests, run inside the checkout
./scripts/check-no-private-deps.sh              # the private module must stay out of the build graph
```

Generator tests never run under the provider's `go test ./...`. Only `make manifestgen-test` runs them. Don't run `go mod tidy` in `tools/manifestgen/`, and don't build it in place: its `go.mod` exists only to keep it out of the root module.

## Pinning and bumping Wayfinder

Rules for `WAYFINDER_REF`:

- It holds a Wayfinder release tag matching `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$` that exists under `refs/tags/`. Branches and bare commit SHAs are rejected.
- Tags older than `v3.2.0` are rejected. That's the first release with the doc metadata the generator needs.
- Automated bumps only move to stable tags (no suffix) that are strictly newer than the current pin. They never downgrade.

To bump by hand:

```sh
make bump-wayfinder REF=v3.4.0
make bump-wayfinder REF=v3.4.0 BUMP_ARGS=--dry-run           # show the plan, change nothing
make bump-wayfinder REF=v3.3.1 BUMP_ARGS=--allow-downgrade   # deliberate downgrade
```

`scripts/bump-wayfinder.sh` refuses to run when `WAYFINDER_REF` or the generated paths (`schema`, `internal/generated`, `docs`, `examples`) have local changes. It then writes `WAYFINDER_REF` and runs `make manifest`, `go generate ./...`, `make docs`, `go build ./...` and `go test ./...`. Last, it runs `go run ./cmd/tfgen -diff <old> <new>` and writes the Terraform schema change summary to `bump-summary.md` (gitignored). If the summary's first line starts with `BREAKING`, the change removes or tightens something users rely on. If any step fails, the script prints the failing step's output and restores `WAYFINDER_REF` and the generated paths.

The scheduled workflow `.github/workflows/bump-wayfinder.yml` runs daily and on manual dispatch (optional `ref` input). It finds the newest stable tag with `scripts/latest-wayfinder-tag.sh`, compares it with `scripts/semver-gt.sh`, and runs the bump script. Then:

- On success it force-pushes branch `chore/wayfinder-bump` and opens or updates one PR titled `chore: bump Wayfinder to <tag>`, with `bump-summary.md` as the body. A `BREAKING` summary adds the `breaking` label.
- On failure it opens or updates one issue titled `Wayfinder <tag>: provider regeneration failed`, with the failing step name, a link to the run and the local reproduce command as the body; diagnostic output is withheld because it can quote private Wayfinder source.

When a bump fails:

1. Read the issue. It names the failing step.
2. Reproduce locally with `make bump-wayfinder REF=<tag>`.
3. Typical causes are a new or removed kind (fix `kinds.yaml`), a renamed or removed field (fix `fields.yaml`), or a schema shape `cmd/tfgen` can't map yet (fix the generator). Commit the fix together with the bump.

When a bump PR is `BREAKING`, review the summary and decide whether the release needs a major version or a migration note before merging.

## Classifying kinds

`tools/manifestgen/kinds.yaml` lists every kind registered at the pinned tag, sorted by group then kind. Each entry has `group`, `kind` and a `class`:

| Class | Result |
|---|---|
| `typed` | A generated resource and data source. |
| `manifest` | Manageable only through `wayfinder_manifest`. |
| `exclude` | Not exposed. `wayfinder_manifest` reports it as "cannot be managed". |

Apply this rule in order:

1. Exclude kinds not served by the API, or System-only kinds.
2. Exclude runtime and record kinds, and kinds the docs say aren't created directly.
3. Exclude read-only kinds.
4. Alpha kinds and user-creatable hidden kinds become `manifest`.
5. Everything else is `typed`.

Optional keys:

- `reason`: required for `exclude` and for `noWait`. Say why, briefly.
- `name: wayfinder_<snake>`: override the Terraform type name. The default is `wayfinder_` plus the snake-cased kind.
- `noWait: true`: don't wait for reconciliation after create or update.

`name` and `noWait` are rejected on excluded kinds. Unknown keys, unsorted entries, duplicates, a registered kind missing from the file and an entry for a kind that no longer exists all fail generation.

The internal kind that stores the variable and secret hierarchy stays excluded. Variables and secrets are managed with the hand-written `wayfinder_variable` and `wayfinder_secret` resources instead; their supported scopes are classified separately in `varScopes`.

### Variable scopes

The top-level `varScopes` list in `tools/manifestgen/kinds.yaml` classifies every scope served by the vars API, sorted by `scope` with no duplicates. Each entry has `scope`, `class: typed|exclude`, and an optional `reason` that is required for `exclude`. `typed` scopes are supported by the hand-written variable and secret resources and recorded in the manifest's `varScopes`; excluded scopes are omitted. The `manifest` class is not valid for variable scopes.

Manifestgen parses non-test Go source files directly in the pinned checkout's `pkg/apiserver/handlers` directory. It reads literal `scopeName` values from functions returning `varsHandler` literals, and counts only constructors called by `apiregister.RegisterHandler(X())` inside `init`. It does not import or execute handler code. Unregistered constructors are ignored; zero registered scopes or a registered constructor with a missing or non-literal scope name fails generation. Scopes served by a different handler type are not detected.

Generation fails if a discovered scope is missing from `varScopes` or a listed scope is no longer served. On a Wayfinder bump, classify each new scope: implement support in `internal/vars` and use `typed`, or use `exclude` with a reason. Update or remove stale entries, regenerate the manifest, and run `make manifestgen-test` and the provider tests. The provider's supported scopes must match the manifest's typed scopes.

## Field overrides

`tools/manifestgen/fields.yaml` carries Terraform-only metadata for single fields. Each entry has `group`, `kind`, `path` and at least one flag:

| Flag | Use it when | Effect |
|---|---|---|
| `immutable: true` | The server rejects changes, or a change only makes sense as a new object. | Changing the value forces replacement. |
| `computed: true` | The server fills the value in when it's unset. | Optional and Computed, with the prior value kept in plan. |
| `ignore: true` | The field can't round-trip, for example a deprecated alias the server clears. Needs `reason`. Can't be combined with other flags. | The field isn't exposed. |

`path` is a dotted JSON path from the object root and must start with `spec.`. Array items are addressed by the array field name, so `spec.rules.verbs` means `verbs` inside each item of `spec.rules`.

Entries are sorted by group, kind, path. Generation fails on duplicates, unsorted entries, unknown keys, a kind that is excluded or missing from `kinds.yaml`, or a path that doesn't exist in the reflected spec. That last check catches stale overrides after a bump: when Wayfinder renames or removes a field, fix or delete its entry. Every entry is also pinned by `internal/generated/fields_pin_test.go`, which fails when an entry is added, removed or changed until its pin table is updated deliberately. The same test checks that each entry lands in the embedded manifest and the generated resource schema.

## How `+encrypted` fields become write-only `_wo` attributes

Wayfinder marks secret fields with `+encrypted`. The generator records them as `sensitive: true` in the manifest, and `cmd/tfgen` turns each one into a pair of attributes:

- `<field>_wo`: write-only and sensitive. Required if the field is required, otherwise optional. It's never stored in plan or state, and Read never sets it.
- `<field>_wo_version`: an optional string stored in state.

Create sends the `_wo` value from configuration. Update sends it again only when `_wo_version` changes. Otherwise the field stays blank, as the server returned it, and the server keeps the stored secret. To rotate a secret, change the value and bump `_wo_version`.

The pair sits at the same level as the original field, so a secret inside a nested object gets its own `_wo` and `_wo_version` inside that object. Data sources omit secret fields entirely.

`cmd/tfgen` fails loudly instead of downgrading to a plain sensitive attribute when a secret can't be modelled this way. That includes a secret below a computed or write-only container, a secret object container, and a secret collection element.

Write-only attributes need Terraform or OpenTofu 1.11 or later.

## Access and secrets

The Wayfinder repository is private. Regenerating the manifest, running `make manifestgen-test` and bumping all need read access to it. Building and testing the provider don't, because they use the committed files.

- Locally, your own git credentials are used against `https://github.com/appvia/wayfinder.git`. With a local clone you can point at it instead. It's still a separate checkout of the pinned tag:

  ```sh
  make manifest WAYFINDER_GIT_URL=file:///path/to/wayfinder
  ```

- In CI, `WAYFINDER_READ_TOKEN` grants read access. The workflows mint it per run with `actions/create-github-app-token` from the appvia org GitHub App (`ORG_ACTIONS_APP_ID` / `ORG_ACTIONS_APP_SECRET`), scoped to `appvia/wayfinder` with contents read; there is no long-lived token. `scripts/wayfinder-git.sh` passes it to git only through `GIT_CONFIG_*` environment variables as an HTTP auth header. It never goes on the command line, in the URL, in `.wayfinder-src/.git/config` or in logs.
- `BUMP_APP_CLIENT_ID` / `BUMP_APP_PRIVATE_KEY` are optional. They belong to a dedicated GitHub App installed only on this repository (Contents, Pull requests and Issues read/write). The bump workflow mints a token from it after regeneration to push the branch and open the PR or failure issue, so that CI runs on the PR. Without them the workflow falls back to `GITHUB_TOKEN`, and you must close and reopen the PR to start checks.

`.github/workflows/manifest.yml` regenerates and checks for drift only on pushes and same-repository PRs. Fork and Dependabot PRs get no secrets, so they skip regeneration. `.github/workflows/test.yml` builds, tests and checks `go generate` drift from the committed files for everyone. Steps that build or run code inside the private Wayfinder checkout run through `scripts/ci-quiet.sh`, which withholds their output from the public log; reproduce a failure locally with the command named in the error. A failed reproducibility check is followed by a step that prints `git status` and the diff of the committed generated paths.

## Adding a new kind

When a Wayfinder release adds a kind, generation fails until it's classified. To add it, or to move an existing kind to another class:

1. Add or edit its entry in `tools/manifestgen/kinds.yaml`, keeping the file sorted by group then kind. Pick the class with the rule above. For `typed`, add `name:` only if the default name reads badly.
2. Add any `fields.yaml` overrides the kind needs (immutable, computed or ignored fields).
3. Check the result without touching `schema/`:

   ```sh
   make manifest MANIFEST_OUT=/tmp/manifest.json
   jq '.kinds[] | select(.kind == "<Kind>") | {kind, typed, terraformName}' /tmp/manifest.json
   ```

4. Regenerate everything and run the tests:

   ```sh
   make manifestgen-test
   make manifest
   make generate
   make docs
   make test
   ```

   `make test` also checks generated descriptions for wording that doesn't suit SaaS users (for example Helm advice). If a kind's Wayfinder docs trip that check, classify it `manifest` and record the reason in `kinds.yaml`.

5. Commit `kinds.yaml`, `fields.yaml` and the regenerated `schema/`, `internal/generated/`, `examples/` and `docs/` together.

## Testing

```sh
make test              # unit tests
make lint
make test-scripts      # shell script scenario tests
make manifestgen-test  # generator tests, needs Wayfinder access
```

### Reproducibility check

```sh
make check-reproducible   # or ./scripts/check-reproducible.sh
```

Deletes `.wayfinder-src/`, refetches the pinned tag, reruns `make manifest`, `go generate ./...` and `make docs`, then fails if `git diff` or `git status --porcelain` shows any change. It refuses to run on a dirty worktree unless `CHECK_REPRODUCIBLE_ALLOW_DIRTY=1`, which compares before/after snapshots instead. CI runs it as the last step on pushes to `main`.

Unit tests run against `internal/fakeapi`, an in-process fake Wayfinder API, so they need no backend.

Acceptance tests live in `internal/acctest` and talk to a real Wayfinder backend. They only run with `TF_ACC=1` set, which `make testacc` does for you. Their setup is described alongside them.

### Running a local build against a real Wayfinder

To try a local build with Terraform against a real backend (for example staging), point Terraform at the built binary with `dev_overrides`. Put the override in a separate CLI config file and select it with `TF_CLI_CONFIG_FILE`, so `~/.terraformrc` stays untouched:

```hcl
# /tmp/wf-dev/dev.tfrc
provider_installation {
  dev_overrides {
    "appvia/wayfinder" = "/path/to/terraform-provider-wayfinder" # directory holding the binary
  }
  direct {}
}
```

`make build` writes `./terraform-provider-wayfinder` at the repo root, so the override points at the repo root. With dev overrides, `terraform plan` and `apply` work without `terraform init`. Terraform prints a warning that overrides are in effect, which is expected. Rerun `make build` after each change; Terraform picks up the new binary on the next command.

For credentials, use your `wf` CLI session. With no credentials configured, the provider reads the profile from `~/.wayfinder/profiles` and refreshes access tokens itself, so nothing expires mid-run while the session lasts:

```sh
wf login                                        # once, against the backend you want
export TF_CLI_CONFIG_FILE=/tmp/wf-dev/dev.tfrc
export WAYFINDER_CLI_PROFILE=staging.wayfinder.run   # the profile to use (wf profiles list)
export WAYFINDER_TENANT=my-tenant               # only if the profile's tenant isn't the one to manage
make build
cd /tmp/wf-dev && terraform plan
```

The server comes from the profile, so leave `WAYFINDER_SERVER` unset (or set it to the profile's server). Make sure `WAYFINDER_TOKEN`, `WAYFINDER_CLIENT_ID`, `WAYFINDER_CLIENT_SECRET` and `WAYFINDER_SERVICE_ACCOUNT` are unset too, since any of them takes precedence over the CLI session.

Keep the scratch configuration outside the repository. Clean up with `terraform destroy`. Prefix sample object names with `tfacc-` so that, if a run is abandoned, `go test ./internal/acctest/... -sweep=ci` removes them. The sweepers only find tenant-scoped objects and objects inside `tfacc-` workspaces; anything in another workspace must be destroyed by hand. Before relying on a result, check that a second `terraform plan` after `apply` reports no changes, and that write-only values do not appear in `terraform.tfstate`.

Acceptance tests (`make testacc`) don't use the CLI session: they still need explicit credentials. Use a service account credential (`wf create serviceaccount NAME --issue-credential`, then `WAYFINDER_CLIENT_ID` and `WAYFINDER_CLIENT_SECRET`).

## Releasing

See [RELEASING.md](RELEASING.md) for the signed release pipeline, required secrets and the pre-release checklist.
