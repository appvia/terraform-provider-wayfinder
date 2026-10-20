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
COPYRIGHT_RE='^(//|#) Copyright 20[0-9]{2} Appvia Ltd\.$'
SPDX_RE='^(//|#) SPDX-License-Identifier: Apache-2\.0$'

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
  # Copyright must be in the first 10 lines; SPDX there too, or directly after the copyright line
  # (where a repair puts it), so a repaired header always passes the next check.
  read -r has_copyright has_spdx < <(CRE="$COPYRIGHT_RE" SRE="$SPDX_RE" awk '
    NR > 11 { exit }
    NR <= 10 && !c && $0 ~ ENVIRON["CRE"] { c = NR }
    (NR <= 10 || (c && NR == c + 1)) && $0 ~ ENVIRON["SRE"] { s = 1 }
    END { print (c ? "true" : "false"), (s ? "true" : "false") }' "$file")
  if $has_copyright && $has_spdx; then
    continue
  fi
  missing=$((missing + 1))
  if $check; then
    if $has_copyright || $has_spdx; then
      echo "incomplete license header: $file" >&2
    else
      echo "missing license header: $file" >&2
    fi
    continue
  fi
  tmp="$(mktemp)"
  if $has_spdx; then
    # Copyright goes directly above the existing SPDX line, in that line's comment style.
    RE="$SPDX_RE" awk -v c="$COPYRIGHT" \
      '!done && NR <= 10 && $0 ~ ENVIRON["RE"] { print substr($0, 1, index($0, " ")) c; done = 1 } { print }' \
      "$file" >"$tmp"
  elif $has_copyright; then
    RE="$COPYRIGHT_RE" awk -v s="$SPDX" \
      '{ print } !done && NR <= 10 && $0 ~ ENVIRON["RE"] { print substr($0, 1, index($0, " ")) s; done = 1 }' \
      "$file" >"$tmp"
  elif [[ "$(head -n 1 "$file")" == '#!'* ]]; then
    block="$prefix $COPYRIGHT"$'\n'"$prefix $SPDX"$'\n'
    { head -n 1 "$file"; printf '%s\n' "$block"; tail -n +2 "$file"; } >"$tmp"
  else
    block="$prefix $COPYRIGHT"$'\n'"$prefix $SPDX"$'\n'
    { printf '%s\n' "$block"; cat "$file"; } >"$tmp"
  fi
  cat "$tmp" >"$file" # keep the file's mode
  rm -f "$tmp"
  echo "fixed license header: $file"
done < <(git ls-files)

if $check && [ "$missing" -gt 0 ]; then
  echo "$missing file(s) with a missing or incomplete license header; run scripts/license-headers.sh" >&2
  exit 1
fi
