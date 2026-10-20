#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Scenario tests for ci-quiet.sh: output withholding, exit status and cleanup.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
failures=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

QUIET="$ROOT/scripts/ci-quiet.sh"
export RUNNER_TEMP="$TMP/runner"
mkdir -p "$RUNNER_TEMP"

# (a) success: exit 0, ok line, private output withheld.
out="$TMP/a.out"
set +e
"$QUIET" "demo" sh -c 'echo PRIVATE-LINE' >"$out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 0 ] && grep -qF 'demo: ok' "$out" && ! grep -qF 'PRIVATE-LINE' "$out"; then
  ok "success: rc 0, 'demo: ok', output withheld"
else bad "success scenario (rc=$rc)"; cat "$out" >&2; fi

# (b) failure: rc preserved, error annotation, private output withheld.
out="$TMP/b.out"
set +e
"$QUIET" "demo" sh -c 'echo PRIVATE-LINE; echo PRIVATE-ERR >&2; exit 3' >"$out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 3 ] && grep -qF '::error::demo failed (exit 3)' "$out" && ! grep -qF 'PRIVATE-' "$out"; then
  ok "failure: rc 3, '::error::demo failed (exit 3)', output withheld"
else bad "failure scenario (rc=$rc)"; cat "$out" >&2; fi

# (c) usage: fewer than two args -> rc 2.
set +e
"$QUIET" "only-label" >"$TMP/c.out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 2 ] && grep -qF 'usage: ci-quiet.sh' "$TMP/c.out"; then
  ok "usage: one arg -> rc 2"
else bad "usage scenario (rc=$rc)"; fi

# (d) cleanup: the log file is removed after success and after failure.
left="$(find "$RUNNER_TEMP" -mindepth 1 | wc -l | tr -d '[:space:]')"
if [ "$left" -eq 0 ]; then ok "cleanup: runner temp dir empty after (a) and (b)"
else bad "cleanup: $left leftover file(s) in runner temp dir"; fi

echo "passed=$pass failed=$failures"
[ "$failures" -eq 0 ]
