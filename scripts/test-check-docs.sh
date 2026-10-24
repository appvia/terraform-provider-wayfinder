#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Scenario tests for check-docs.sh against throwaway docs/examples trees in a temp dir.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK="$ROOT/scripts/check-docs.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
failures=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

mkdir -p "$TMP/docs/resources" "$TMP/docs/data-sources" "$TMP/docs/ephemeral-resources" \
  "$TMP/examples/ephemeral-resources/wayfinder_webhook_ingest_url"
for page in resources/workspace data-sources/workspace ephemeral-resources/webhook_ingest_url; do
  printf '# page\n\n## Example Usage\n' >"$TMP/docs/$page.md"
done
example="$TMP/examples/ephemeral-resources/wayfinder_webhook_ingest_url/ephemeral-resource.tf"
printf 'ephemeral "wayfinder_webhook_ingest_url" "x" {}\n' >"$example"

# (a) every page has its section and the ephemeral example exists: rc 0, silent.
set +e
"$CHECK" "$TMP/docs" "$TMP/examples" >"$TMP/a.out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 0 ] && [ ! -s "$TMP/a.out" ]; then
  ok "complete docs: rc 0, no output"
else bad "complete docs scenario (rc=$rc)"; cat "$TMP/a.out" >&2; fi

# (b) ephemeral example removed: rc 1 and names the missing file.
rm "$example"
set +e
"$CHECK" "$TMP/docs" "$TMP/examples" >"$TMP/b.out" 2>&1
rc=$?
set -e
want="check-docs: $TMP/docs/ephemeral-resources/webhook_ingest_url.md has no example at $example"
if [ "$rc" -eq 1 ] && grep -qxF "$want" "$TMP/b.out"; then
  ok "missing ephemeral example: rc 1, names the example"
else bad "missing ephemeral example scenario (rc=$rc)"; cat "$TMP/b.out" >&2; fi

# (c) ephemeral page without an Example Usage section: rc 1 and names the page.
printf 'ephemeral "wayfinder_webhook_ingest_url" "x" {}\n' >"$example"
printf '# page\n' >"$TMP/docs/ephemeral-resources/webhook_ingest_url.md"
set +e
"$CHECK" "$TMP/docs" "$TMP/examples" >"$TMP/c.out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 1 ] && grep -qF 'webhook_ingest_url.md has no "## Example Usage" section' "$TMP/c.out"; then
  ok "ephemeral page without example section: rc 1"
else bad "ephemeral section scenario (rc=$rc)"; cat "$TMP/c.out" >&2; fi

echo "passed=$pass failed=$failures"
[ "$failures" -eq 0 ]
