#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Usage: bump-wayfinder.sh <tag> [--dry-run] [--allow-downgrade]
#
# Moves WAYFINDER_REF to <tag>, regenerates the manifest, code and docs, and
# runs build + tests (all gates run here: a PR opened with GITHUB_TOKEN gets no
# CI). Writes the Terraform schema change summary to bump-summary.md.
# On any failure the generated paths and WAYFINDER_REF are restored.
#
# Test hook: BUMP_TEST_FAIL_AFTER_GENERATE=1 creates docs/resources/zz_new.md
# after `go generate` and then fails (exercises the restore path).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/wayfinder-git.sh
. "$ROOT/scripts/wayfinder-git.sh"

GENERATED="schema internal/generated docs examples"
TAG_RE='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'
MIN_REF="${WAYFINDER_MIN_REF:-v3.2.0}"

usage() {
  echo "usage: $0 <tag> [--dry-run] [--allow-downgrade]" >&2
  exit 2
}

tag=""
dry_run=0
allow_downgrade=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) dry_run=1 ;;
    --allow-downgrade) allow_downgrade=1 ;;
    -*) usage ;;
    *)
      [ -z "$tag" ] || usage
      tag="$arg"
      ;;
  esac
done
[ -n "$tag" ] || usage

# --- validate (D-PIN) ---
if ! printf '%s\n' "$tag" | grep -Eq "$TAG_RE"; then
  echo "WAYFINDER_REF must be a Wayfinder release tag like v3.3.2" >&2
  exit 1
fi
if "$ROOT/scripts/semver-gt.sh" "$MIN_REF" "$tag"; then
  echo "WAYFINDER_REF $tag is older than the supported minimum $MIN_REF" >&2
  exit 1
fi
current="$(tr -d '[:space:]' < "$ROOT/WAYFINDER_REF")"
if [ "$tag" = "$current" ]; then
  echo "already at $tag"
  exit 0
fi
if ! "$ROOT/scripts/semver-gt.sh" "$tag" "$current"; then
  if [ "$allow_downgrade" -ne 1 ]; then
    echo "refusing to downgrade from $current to $tag" >&2
    exit 1
  fi
  echo "warning: downgrading from $current to $tag (--allow-downgrade)" >&2
fi

# --- resolve (peeled commits) ---
peeled_commit() { # <tag>
  wf_git ls-remote --tags "$WF_URL" "refs/tags/$1" "refs/tags/$1^{}" | awk -v r="refs/tags/$1" '
    $2 == r "^{}" { peeled = $1 } $2 == r { plain = $1 }
    END { print (peeled != "" ? peeled : plain) }'
}
target_commit="$(peeled_commit "$tag")"
if [ -z "$target_commit" ]; then
  echo "tag $tag not found in $(wf_redacted_url)" >&2
  exit 1
fi
current_commit="$(peeled_commit "$current")"

if [ "$dry_run" -eq 1 ]; then
  echo "bump plan (dry run, nothing changed):"
  echo "  current: $current (${current_commit:-commit not found})"
  echo "  target:  $tag ($target_commit)"
  echo "  steps:   write WAYFINDER_REF; make manifest; go generate ./...; make docs;"
  echo "           go build ./...; go test ./...; tfgen -diff > bump-summary.md"
  exit 0
fi

# --- preflight ---
# shellcheck disable=SC2086 # GENERATED is a fixed list of path arguments
if [ -n "$(git status --porcelain -- WAYFINDER_REF $GENERATED)" ]; then
  echo "generated paths have local changes; commit or discard them first:" >&2
  # shellcheck disable=SC2086
  git status --short -- WAYFINDER_REF $GENERATED >&2
  exit 1
fi

TMP="$(mktemp -d)"
restore() {
  local rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "bump-wayfinder: restoring WAYFINDER_REF and generated files" >&2
    local path
    for path in WAYFINDER_REF $GENERATED; do
      if [ -n "$(git ls-files -- "$path")" ]; then
        git checkout -- "$path"
      fi
    done
    # shellcheck disable=SC2086
    git clean -fdq -- $GENERATED
  fi
  rm -rf "$TMP"
  exit "$rc"
}
trap restore EXIT

cp schema/wayfinder-manifest.json "$TMP/old-manifest.json"

step() { # <name> <command...>; output kept quiet unless the step fails
  local name="$1"
  shift
  echo "==> $name"
  if ! "$@" > "$TMP/step.log" 2>&1; then
    {
      echo "bump to $tag failed at step: $name"
      echo "--- output (last 200 lines) ---"
      tail -n 200 "$TMP/step.log"
    } >&2
    exit 1
  fi
}

printf '%s\n' "$tag" > WAYFINDER_REF
step "make manifest" make manifest
step "go generate ./..." go generate ./...
if [ "${BUMP_TEST_FAIL_AFTER_GENERATE:-}" = 1 ]; then
  mkdir -p docs/resources
  echo "simulated generated file" > docs/resources/zz_new.md
  step "test hook BUMP_TEST_FAIL_AFTER_GENERATE" false
fi
step "make docs" make docs
step "go build ./..." go build ./...
step "go test ./..." go test ./... -count=1
summarize() { go run ./cmd/tfgen -diff "$TMP/old-manifest.json" schema/wayfinder-manifest.json > "$TMP/summary.md"; }
step "tfgen -diff" summarize
mv "$TMP/summary.md" bump-summary.md

echo "bumped Wayfinder $current ($current_commit) -> $tag ($target_commit); summary in bump-summary.md"
