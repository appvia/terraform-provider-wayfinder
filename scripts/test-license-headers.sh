#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Scenario tests for license-headers.sh: check mode, insertion, shebangs and idempotence.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
failures=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

HEADERS="$ROOT/scripts/license-headers.sh"
YEAR="$(date +%Y)"
cd "$TMP"
git init -q .
printf 'package main\n' >main.go
printf '#!/usr/bin/env bash\necho hi\n' >run.sh
chmod +x run.sh
printf '# Title\n' >README.md
mkdir -p x/testdata
printf 'package fixture\n' >x/testdata/f.go
git add .

# (a) check mode: rc 1, names the unheadered files, changes nothing.
set +e
"$HEADERS" --check >"$TMP/a.out" 2>&1
rc=$?
set -e
if [ "$rc" -eq 1 ] && grep -qF 'missing license header: main.go' "$TMP/a.out" &&
  grep -qF 'missing license header: run.sh' "$TMP/a.out" && ! grep -qF 'README.md' "$TMP/a.out" &&
  ! grep -qF 'testdata' "$TMP/a.out" && [ "$(cat main.go)" = 'package main' ]; then
  ok "check: rc 1, lists main.go and run.sh only, no change"
else bad "check scenario (rc=$rc)"; cat "$TMP/a.out" >&2; fi

# (b) fix mode: Go header on top with a blank line, shell header after the shebang, mode kept.
"$HEADERS" >/dev/null
want_go="// Copyright $YEAR Appvia Ltd."$'\n'"// SPDX-License-Identifier: Apache-2.0"$'\n\n'"package main"
want_sh="#!/usr/bin/env bash"$'\n'"# Copyright $YEAR Appvia Ltd."$'\n'"# SPDX-License-Identifier: Apache-2.0"$'\n\n'"echo hi"
if [ "$(cat main.go)" = "$want_go" ] && [ "$(cat run.sh)" = "$want_sh" ] && [ -x run.sh ] &&
  [ "$(cat README.md)" = '# Title' ] && [ "$(cat x/testdata/f.go)" = 'package fixture' ]; then
  ok "fix: headers inserted, shebang first, mode kept, md/testdata untouched"
else bad "fix scenario"; cat main.go run.sh >&2; fi

# (c) idempotent: second run adds nothing and check passes.
before="$(cat main.go run.sh)"
"$HEADERS" >"$TMP/c.out"
if [ ! -s "$TMP/c.out" ] && [ "$(cat main.go run.sh)" = "$before" ] && "$HEADERS" --check; then
  ok "idempotent: rerun is a no-op, check rc 0"
else bad "idempotent scenario"; fi

echo "passed=$pass failed=$failures"
[ "$failures" -eq 0 ]
