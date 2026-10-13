#!/usr/bin/env bash
# Guards that the provider module never depends on the private Wayfinder module.
set -euo pipefail

PRIV="github.com/appvia/wayfinder"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail=0
err() { echo "FAIL: $*" >&2; fail=1; }

# (a) build graph
if go list -deps ./... 2>/dev/null | grep -E "^${PRIV}(/|$)" >/dev/null; then
  err "(a) go list -deps ./... contains ${PRIV}"
fi

# (b) *.go outside tools/manifestgen must not import the private module (even behind a build
# tag, which (a) cannot see). Import-shaped lines only: generated API descriptions may quote
# the repository name as an example value.
IMPORT_RE='^[[:space:]]*(import[[:space:]]+)?([A-Za-z_.][A-Za-z0-9_]*[[:space:]]+)?"github\.com/appvia/wayfinder(/[^"]*)?"'
while IFS= read -r f; do
  if grep -E "$IMPORT_RE" "$f" >/dev/null 2>&1; then
    err "(b) ${f} references ${PRIV}"
  fi
done < <(find . \
  \( -path ./.wayfinder-src -o -path ./.git -o -path ./.omo -o -path ./.codegraph -o -path ./tools/manifestgen \) -prune \
  -o -type f -name '*.go' -print)

# (c) every .go under tools/manifestgen starts with the build tag
has_gen_go=0
if [ -d tools/manifestgen ]; then
  while IFS= read -r f; do
    has_gen_go=1
    first="$(head -n 1 "$f")"
    if [ "$first" != "//go:build wfmanifestgen" ]; then
      err "(c) ${f} must start with '//go:build wfmanifestgen'"
    fi
  done < <(find tools/manifestgen -type f -name '*.go' -print)
fi

# (d) nested module required
if [ "$has_gen_go" -eq 1 ] && [ ! -f tools/manifestgen/go.mod ]; then
  err "(d) tools/manifestgen has .go files but no tools/manifestgen/go.mod"
fi

# (e) go mod tidy must not change go.mod/go.sum nor add the private module
hash_files() {
  local f
  for f in go.mod go.sum; do
    if [ -f "$f" ]; then cksum < "$f"; else echo "absent"; fi
  done
}
before="$(hash_files)"
go mod tidy >/dev/null 2>&1 || err "(e) go mod tidy failed"
after="$(hash_files)"
if [ "$before" != "$after" ]; then
  err "(e) go mod tidy changed go.mod/go.sum; run it and commit the result"
fi
if [ -f go.sum ] && grep -F "appvia/wayfinder" go.mod go.sum 2>/dev/null | grep -Ev "terraform-provider-wayfinder" >/dev/null; then
  err "(e) go.mod/go.sum mention appvia/wayfinder"
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: no private Wayfinder dependencies"
