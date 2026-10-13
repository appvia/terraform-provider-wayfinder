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

<!-- Add further sections below this one. Keep the "Manifest contract" heading as is. -->
