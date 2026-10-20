#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Usage: manifestgen.sh run|test [extra generator args]
# Compiles tools/manifestgen INSIDE the pinned Wayfinder checkout so Wayfinder's
# own go.mod/go.sum govern the build (see DEVELOPMENT.md).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [ "$#" -lt 1 ]; then
  echo "usage: $0 run|test [args...]" >&2
  exit 2
fi
MODE="$1"
shift

checkout_out="$("$ROOT/scripts/wayfinder-checkout.sh")"
printf '%s\n' "$checkout_out"
SHA="$(printf '%s\n' "$checkout_out" | tail -n 1 | sed -n 's/^WAYFINDER_COMMIT=//p')"
REF="$(tr -d '[:space:]' < "$ROOT/WAYFINDER_REF")"
if [ -z "$SHA" ]; then
  echo "wayfinder-checkout.sh did not report WAYFINDER_COMMIT" >&2
  exit 1
fi

SRC_DIR="$ROOT/.wayfinder-src"
GEN_DIR="$SRC_DIR/hack/tfprovider-manifestgen"
rm -rf "$GEN_DIR"
mkdir -p "$GEN_DIR"
for f in "$ROOT"/tools/manifestgen/*.go; do
  [ -e "$f" ] && cp "$f" "$GEN_DIR/"
done
if [ -d "$ROOT/tools/manifestgen/testdata" ]; then
  cp -R "$ROOT/tools/manifestgen/testdata" "$GEN_DIR/testdata"
fi
if [ -f "$ROOT/internal/manifest/contract.go" ]; then
  mkdir -p "$GEN_DIR/manifest"
  cp "$ROOT/internal/manifest/contract.go" "$GEN_DIR/manifest/contract.go"
fi

export GOTOOLCHAIN=auto
KINDS="$ROOT/tools/manifestgen/kinds.yaml"
FIELDS="$ROOT/tools/manifestgen/fields.yaml"

case "$MODE" in
  run)
    OUT="${MANIFEST_OUT:-$ROOT/schema/wayfinder-manifest.json}"
    mkdir -p "$(dirname "$OUT")"
    cd "$SRC_DIR"
    go run -tags wfmanifestgen ./hack/tfprovider-manifestgen \
      -kinds "$KINDS" -fields "$FIELDS" \
      -wayfinder-version "$REF" -wayfinder-commit "$SHA" \
      -o "$OUT" "$@"
    ;;
  test)
    export MANIFESTGEN_KINDS="$KINDS" MANIFESTGEN_FIELDS="$FIELDS"
    cd "$SRC_DIR"
    go test -tags wfmanifestgen ./hack/tfprovider-manifestgen/... "$@"
    ;;
  *)
    echo "unknown mode '$MODE'; want run|test" >&2
    exit 2
    ;;
esac
