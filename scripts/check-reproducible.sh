#!/usr/bin/env bash
# Clean-room reproducibility check: refetch the pinned Wayfinder tag, regenerate
# manifest, Go code and docs, and fail if anything differs from what is committed.
# CHECK_REPRODUCIBLE_ALLOW_DIRTY=1 permits a dirty start; the before/after
# snapshots of the tree are compared instead.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

hash_stdin() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum; else shasum -a 256; fi
}

# Snapshot = porcelain status + content hash of every modified/untracked file.
snapshot() {
  git status --porcelain
  git ls-files -m -o --exclude-standard -z | while IFS= read -r -d '' f; do
    if [ -f "$f" ]; then
      printf '%s  %s\n' "$(hash_stdin < "$f" | awk '{print $1}')" "$f"
    else
      printf 'missing  %s\n' "$f"
    fi
  done
}

allow_dirty="${CHECK_REPRODUCIBLE_ALLOW_DIRTY:-0}"
if [ -n "$(git status --porcelain)" ] && [ "$allow_dirty" != "1" ]; then
  echo "check-reproducible: worktree has uncommitted changes; the check would be meaningless." >&2
  echo "Commit or stash them, or set CHECK_REPRODUCIBLE_ALLOW_DIRTY=1 to compare before/after." >&2
  git status --short >&2
  exit 1
fi

before="$(mktemp)"
after="$(mktemp)"
trap 'rm -f "$before" "$after"' EXIT
snapshot > "$before"

rm -rf "$ROOT/.wayfinder-src"
make manifest
go generate ./...
make docs

snapshot > "$after"

if [ "$allow_dirty" = "1" ]; then
  if ! diff -u "$before" "$after"; then
    echo "check-reproducible: FAIL: regeneration changed the tree (diff above)" >&2
    exit 1
  fi
else
  if ! git diff --exit-code; then
    echo "check-reproducible: FAIL: regenerated output differs from committed files" >&2
    exit 1
  fi
  if [ -n "$(git status --porcelain)" ]; then
    echo "check-reproducible: FAIL: untracked or changed files after regeneration:" >&2
    git status --short >&2
    exit 1
  fi
fi

tag="$(tr -d '[:space:]' < "$ROOT/WAYFINDER_REF")"
commit="$(git -C "$ROOT/.wayfinder-src" rev-parse HEAD)"
echo "check-reproducible: OK tag=$tag commit=$commit"
