# internal/engine

## OVERVIEW
One generic `kindResource` / `kindDataSource` implements CRUD, import, readiness waiting and API-error diagnostics for every typed kind; generated code only supplies `manifest.Kind` + framework schema.

## WHERE TO LOOK
| Task | File | Notes |
|------|------|-------|
| Construction, type name, Configure | resource.go | `prepared()` guards every op (mapping error / nil client) |
| Create | create.go | `writeValues` → `convert.ToObject` → POST → store generation → `FromObject` → optional wait |
| Read / drift | read.go | No baseline (fresh import) → `PopulateAll`, then stores generation |
| Update | update.go | Fresh GET, generation check, overlay config on fresh object (`ToOpts.Base`, `Prior`) |
| Delete | delete.go | DELETE (404 = already gone), then polls GET until 404 |
| Import ID parsing | importstate.go | `convert.ParseImportID`: `[workspace/[environment/]]name[@version]` |
| `_wo` selection | secrets.go | Create sends all set `_wo`; Update only when `_wo_version` changed |
| Timeouts (30m default), wait toggle | timeouts.go | Waits only if `HasStatus && WaitForReady && wait_for_ready` |
| Readiness polling | wait.go | 2s doubling to 15s; reconciled = generation + operationID match |
| Error → diagnostics | errors.go | `AddAPIError`: import hints for exists/modified, per-field validation paths |
| Data source | datasource.go | Reuses `kindResource.prepared`; projects onto data-source schema (no secrets) |

## CONVENTIONS
- Exported surface reused by `manifestresource`: `Waiter`, `Poll`, `DefaultPoll`, `AddAPIError`, `ErrorTarget`, `Reference`, `Generation`. Keep them stable.
- Private state key is literally `"generation"`; a mismatch with the fresh GET → `client.ErrModified` ("changed outside Terraform").
- Status mapping: Success/Complete ok; Warning/ActionRequired/ApprovalRequired → warning; Failure/Error → error.
- Tests use `newResource(..., pollOptions{...})` with tiny intervals against `fakeapi`; never sleep on `DefaultPoll` in tests.

## ANTI-PATTERNS
- Kind-specific branches here. Per-kind behaviour belongs in the manifest (`kinds.yaml`/`fields.yaml`) or `convert`.
- Sending a full object built from state on Update; always overlay onto the fresh GET so unmanaged/server fields survive.
- Setting `_wo` attributes in state, or re-sending secrets without a `_wo_version` change.
