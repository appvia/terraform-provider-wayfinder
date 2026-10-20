#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Usage: license-headers.sh [--check]
# Adds the SPDX copyright header to every tracked source file in the current git repo that lacks
# one. With --check, changes nothing and exits 1 listing the files without a header.
# Go headers are also enforced by golangci-lint (goheader); tfgen writes the header into
# generated files itself.
set -euo pipefail

SPDX='SPDX-License-Identifier: Apache-2.0'
COPYRIGHT="Copyright $(date +%Y) Appvia Ltd."

check=false
case "${1:-}" in
  --check) check=true ;;
  "") ;;
  *) echo "usage: license-headers.sh [--check]" >&2; exit 2 ;;
esac

# Prints the comment prefix for a file, or nothing when the file takes no header.
prefix_for() {
  case "$1" in
    */testdata/* | docs/* | templates/*) ;;
    *.go) echo '//' ;;
    *.sh | *.tf | *.yml | *.yaml | *.toml | Makefile | .gitignore | .shellcheckrc) echo '#' ;;
  esac
}

missing=0
while IFS= read -r file; do
  prefix="$(prefix_for "$file")"
  [ -n "$prefix" ] || continue
  [ -f "$file" ] || continue
  top="$(head -n 10 "$file")"
  if [[ "$top" == *"$SPDX"* ]]; then
    continue
  fi
  missing=$((missing + 1))
  if $check; then
    echo "missing license header: $file" >&2
    continue
  fi
  block="$prefix $COPYRIGHT"$'\n'"$prefix $SPDX"$'\n'
  tmp="$(mktemp)"
  if [[ "$top" == '#!'* ]]; then
    { head -n 1 "$file"; printf '%s\n' "$block"; tail -n +2 "$file"; } >"$tmp"
  else
    { printf '%s\n' "$block"; cat "$file"; } >"$tmp"
  fi
  cat "$tmp" >"$file" # keep the file's mode
  rm -f "$tmp"
  echo "added license header: $file"
done < <(git ls-files)

if $check && [ "$missing" -gt 0 ]; then
  echo "$missing file(s) missing a license header; run scripts/license-headers.sh" >&2
  exit 1
fi
