#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Shallow checkout of exactly the release tag in WAYFINDER_REF into .wayfinder-src/.
# Last stdout line: WAYFINDER_COMMIT=<peeled 40-hex commit sha>.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/wayfinder-git.sh
. "$ROOT/scripts/wayfinder-git.sh"

SRC_DIR="$ROOT/.wayfinder-src"
MIN_REF="${WAYFINDER_MIN_REF:-v3.2.0}"

REF="$(tr -d '[:space:]' < "$ROOT/WAYFINDER_REF")"
if ! printf '%s\n' "$REF" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'; then
  echo "WAYFINDER_REF must be a Wayfinder release tag like v3.3.2" >&2
  exit 1
fi
if "$ROOT/scripts/semver-gt.sh" "$MIN_REF" "$REF"; then
  echo "WAYFINDER_REF $REF is older than the supported minimum $MIN_REF" >&2
  exit 1
fi

# Peeled commit: use the ^{} line (annotated tag) when present, else the plain line.
lines="$(wf_git ls-remote --tags "$WF_URL" "refs/tags/$REF" "refs/tags/$REF^{}")"
SHA="$(printf '%s\n' "$lines" | awk -v r="refs/tags/$REF" '
  $2 == r "^{}" { peeled = $1 } $2 == r { plain = $1 }
  END { print (peeled != "" ? peeled : plain) }')"
if [ -z "$SHA" ]; then
  echo "tag $REF not found in $(wf_redacted_url)" >&2
  exit 1
fi

if [ -d "$SRC_DIR/.git" ] && [ "$(git -C "$SRC_DIR" rev-parse HEAD 2>/dev/null || true)" = "$SHA" ]; then
  echo "reusing $SRC_DIR at $REF ($SHA)"
else
  echo "fetching $REF from $(wf_redacted_url) into $SRC_DIR"
  rm -rf "$SRC_DIR"
  git init -q "$SRC_DIR"
  (cd "$SRC_DIR" && wf_git fetch -q --depth 1 --no-tags "$WF_URL" "refs/tags/$REF:refs/tags/$REF")
  git -C "$SRC_DIR" checkout -q --detach "refs/tags/$REF^{commit}"
  head="$(git -C "$SRC_DIR" rev-parse HEAD)"
  if [ "$head" != "$SHA" ]; then
    echo "checkout HEAD $head does not match tag $REF commit $SHA" >&2
    exit 1
  fi
fi

echo "WAYFINDER_COMMIT=$SHA"
