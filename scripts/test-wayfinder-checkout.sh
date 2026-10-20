#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Scenario tests for wayfinder-checkout.sh, latest-wayfinder-tag.sh and semver-gt.sh.
# Builds a throwaway git repo with release tags and runs the scripts from a
# temporary copy of the repo root, so the real .wayfinder-src is never touched.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Isolate git from the user's global/system config (signing, hooks, ...).
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

pass=0
failures=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

# --- Given: a source repo with lightweight + annotated tags and a branch ---
SRC="$TMP/src"
git init -q "$SRC"
commit_tag() { # <tag> [annotated]
  git -C "$SRC" commit -q --allow-empty -m "release $1"
  if [ "${2:-}" = annotated ]; then
    git -C "$SRC" tag -a "$1" -m "annotated $1"
  else
    git -C "$SRC" tag "$1"
  fi
}
commit_tag v0.9.0
commit_tag v1.0.0
commit_tag v1.2.0
commit_tag v1.3.0 annotated
commit_tag v1.10.0
commit_tag v1.11.0-rc1
git -C "$SRC" branch develop

WORK="$TMP/root"
mkdir -p "$WORK/scripts"
cp "$ROOT"/scripts/*.sh "$WORK/scripts/"
export WAYFINDER_GIT_URL="file://$SRC" WAYFINDER_MIN_REF=v1.0.0
unset WAYFINDER_READ_TOKEN || true

OUT="$TMP/out"
run_checkout() { # <ref>; sets rc, output in $OUT
  printf '%s\n' "$1" > "$WORK/WAYFINDER_REF"
  set +e
  "$WORK/scripts/wayfinder-checkout.sh" > "$OUT" 2>&1
  rc=$?
  set -e
}
last_commit() { tail -n 1 "$OUT" | sed -n 's/^WAYFINDER_COMMIT=//p'; }

# --- lightweight tag ---
want="$(git -C "$SRC" rev-parse 'v1.2.0^{commit}')"
run_checkout v1.2.0
if [ "$rc" -eq 0 ] && [ "$(last_commit)" = "$want" ] \
  && [ "$(git -C "$WORK/.wayfinder-src" rev-parse HEAD)" = "$want" ]; then
  ok "lightweight tag v1.2.0 -> $want"
else bad "lightweight tag v1.2.0 (rc=$rc)"; cat "$OUT" >&2; fi

# --- second run reuses without fetching ---
run_checkout v1.2.0
if [ "$rc" -eq 0 ] && grep -q reusing "$OUT" && ! grep -q fetching "$OUT" \
  && [ "$(last_commit)" = "$want" ]; then
  ok "second run reuses checkout"
else bad "second run reuse (rc=$rc)"; cat "$OUT" >&2; fi

# --- annotated tag (peeled) ---
want="$(git -C "$SRC" rev-parse 'v1.3.0^{commit}')"
tagobj="$(git -C "$SRC" rev-parse v1.3.0)"
run_checkout v1.3.0
if [ "$rc" -eq 0 ] && [ "$(last_commit)" = "$want" ] && [ "$want" != "$tagobj" ] \
  && [ "$(git -C "$WORK/.wayfinder-src" rev-parse HEAD)" = "$want" ]; then
  ok "annotated tag v1.3.0 -> peeled $want"
else bad "annotated tag v1.3.0 (rc=$rc)"; cat "$OUT" >&2; fi

# --- rejections ---
expect_fail() { # <ref> <message> <label>
  run_checkout "$1"
  if [ "$rc" -ne 0 ] && grep -qF -- "$2" "$OUT"; then ok "$3"
  else bad "$3 (rc=$rc)"; cat "$OUT" >&2; fi
}
expect_fail develop "WAYFINDER_REF must be a Wayfinder release tag like v3.3.2" "branch develop rejected"
expect_fail "$(git -C "$SRC" rev-parse v1.0.0)" "WAYFINDER_REF must be a Wayfinder release tag like v3.3.2" "40-hex SHA rejected"
expect_fail v0.9.0 "WAYFINDER_REF v0.9.0 is older than the supported minimum v1.0.0" "v0.9.0 older than minimum"
expect_fail v9.9.9 "tag v9.9.9 not found in file://$SRC" "v9.9.9 not found"

# --- token never persisted or printed ---
rm -rf "$WORK/.wayfinder-src"
export WAYFINDER_READ_TOKEN=dummy-token-123
token_b64="$(printf 'x-access-token:%s' "$WAYFINDER_READ_TOKEN" | base64 | tr -d '\n')"
run_checkout v1.10.0
unset WAYFINDER_READ_TOKEN
if [ "$rc" -eq 0 ] \
  && ! grep -rqF -e dummy-token-123 -e "$token_b64" "$WORK/.wayfinder-src/.git" \
  && ! grep -qF -e dummy-token-123 -e "$token_b64" "$OUT"; then
  ok "token absent from .wayfinder-src/.git and output"
else bad "token leak check (rc=$rc)"; cat "$OUT" >&2; fi
# Pattern assembled at runtime so this file does not match itself.
argv_pat="-c"
argv_pat="$argv_pat http"
if grep -n -- "$argv_pat" "$ROOT"/scripts/*.sh; then bad "scripts pass git config via argv"
else ok "no git config argv usage"; fi

# --- latest stable tag ---
latest="$("$WORK/scripts/latest-wayfinder-tag.sh")"
if [ "$latest" = v1.10.0 ]; then ok "latest stable = v1.10.0"
else bad "latest stable = '$latest', want v1.10.0"; fi

# --- semver-gt ---
semver_case() { # <a> <b> <want-rc>
  set +e; "$WORK/scripts/semver-gt.sh" "$1" "$2"; local got=$?; set -e
  if [ "$got" -eq "$3" ]; then ok "semver-gt $1 $2 -> $3"
  else bad "semver-gt $1 $2 -> $got, want $3"; fi
}
semver_case v1.10.0 v1.2.0 0
semver_case v1.2.0 v1.10.0 1
semver_case v1.11.0-rc1 v1.11.0 1
semver_case v1.11.0 v1.11.0-rc1 0
semver_case v1.2.0 v1.2.0 1
semver_case v2.0.0 v1.99.99 0

echo "passed=$pass failed=$failures"
[ "$failures" -eq 0 ]
