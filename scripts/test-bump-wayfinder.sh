#!/usr/bin/env bash
# Scenario tests for bump-wayfinder.sh failure, restore and preflight paths.
#
# Runs every scenario in a throwaway clone of this repository (HEAD plus the
# working-tree copies of scripts/, Makefile and cmd/tfgen/), against a
# synthetic file:// Wayfinder source that carries the pinned commit under the
# extra tags v3.3.1 and v3.3.3. The cached .wayfinder-src is copied into the
# clone so no network access is needed. The real working tree is never touched.
#
# Set BUMP_TEST_SUCCESS=1 to also run a full successful bump (slow: go test).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_TERMINAL_PROMPT=0
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com
unset WAYFINDER_READ_TOKEN BUMP_TEST_FAIL_AFTER_GENERATE || true

pass=0
failures=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }
check() { # <label> <command...>
  local label="$1"
  shift
  if "$@"; then ok "$label"; else bad "$label"; fi
}

pin="$(tr -d '[:space:]' < "$ROOT/WAYFINDER_REF")"
if [ ! -d "$ROOT/.wayfinder-src/.git" ]; then
  echo "run 'make wayfinder-src' first (needs the cached $pin checkout)" >&2
  exit 1
fi

# --- Given: synthetic Wayfinder source with the pinned commit under 3 tags ---
SRC="$TMP/wayfinder"
git init -q "$SRC"
git -C "$SRC" fetch -q --update-shallow "$ROOT/.wayfinder-src" "refs/tags/$pin:refs/tags/$pin"
git -C "$SRC" tag v3.3.1 "$pin"
git -C "$SRC" tag v3.3.3 "$pin"
pin_commit="$(git -C "$SRC" rev-parse "$pin^{commit}")"
export WAYFINDER_GIT_URL="file://$SRC"

# --- Given: a throwaway clone of the provider repo ---
C="$TMP/provider"
git clone -q "$ROOT" "$C"
rm -rf "$C/scripts" "$C/cmd/tfgen"
cp -R "$ROOT/scripts" "$C/scripts"
cp -R "$ROOT/cmd/tfgen" "$C/cmd/tfgen"
cp "$ROOT/Makefile" "$C/Makefile"
git -C "$C" add -A
git -C "$C" commit -qm "bump test overlay" --allow-empty
cp -R "$ROOT/.wayfinder-src" "$C/.wayfinder-src"

OUT="$TMP/out"
bump() { # <args...>; sets rc, output in $OUT
  set +e
  (cd "$C" && ./scripts/bump-wayfinder.sh "$@") > "$OUT" 2>&1
  rc=$?
  set -e
}
clean_tree() { [ -z "$(git -C "$C" status --porcelain)" ]; }
expect() { # <label> <want-rc: 0|nonzero> <message>
  local rc_ok=1
  if [ "$2" = 0 ]; then [ "$rc" -eq 0 ] || rc_ok=0; else [ "$rc" -ne 0 ] || rc_ok=0; fi
  if [ "$rc_ok" -eq 1 ] && grep -qF -- "$3" "$OUT"; then ok "$1"
  else bad "$1 (rc=$rc)"; cat "$OUT" >&2; fi
}

# --- validation ---
bump develop
expect "branch develop rejected (D-PIN)" nonzero "WAYFINDER_REF must be a Wayfinder release tag like v3.3.2"
check "tree clean after develop" clean_tree
bump v3.2.0
expect "older tag refused" nonzero "refusing to downgrade from $pin to v3.2.0"
bump "$pin"
expect "same tag is a no-op" 0 "already at $pin"
bump v3.3.3 --dry-run
expect "dry-run names current and target" 0 "current: $pin ($pin_commit)"
check "dry-run names target commit" grep -qF "target:  v3.3.3 ($pin_commit)" "$OUT"
check "tree clean after dry-run" clean_tree

# --- generator failure (kind missing from kinds.yaml) restores everything ---
kinds="$C/tools/manifestgen/kinds.yaml"
awk 'BEGIN { skip = 0; done = 0 }
  /^  - group:/ { if (!done && seen++) { skip = 1; done = 1 } else { skip = 0 } }
  !skip' "$kinds" > "$TMP/kinds.yaml"
cp "$TMP/kinds.yaml" "$kinds"
bump v3.3.1 --allow-downgrade
expect "missing kind fails at make manifest" nonzero "failed at step: make manifest"
if [ "$(git -C "$C" status --porcelain)" = " M tools/manifestgen/kinds.yaml" ]; then
  ok "only the scratch kinds.yaml remains modified"
else bad "unexpected changes: $(git -C "$C" status --porcelain)"; fi
git -C "$C" checkout -- tools/manifestgen/kinds.yaml
check "tree clean after kinds.yaml restore" clean_tree

# --- failure after generate with docs/ absent (untracked-only) ---
# docs/ is committed in the real repo; this scenario needs a clone where it is absent.
if git -C "$C" ls-files --error-unmatch docs >/dev/null 2>&1; then
  git -C "$C" rm -rq docs
  git -C "$C" commit -qm "drop tracked docs for the docs-absent scenario"
fi
[ ! -e "$C/docs" ] || bad "precondition: docs/ should be absent at HEAD"
export BUMP_TEST_FAIL_AFTER_GENERATE=1
bump v3.3.3
expect "hook failure (docs absent)" nonzero "failed at step: test hook BUMP_TEST_FAIL_AFTER_GENERATE"
if clean_tree && [ ! -e "$C/docs" ] && [ "$(cat "$C/WAYFINDER_REF")" = "$pin" ]; then
  ok "restored: tree clean, docs/ removed, WAYFINDER_REF=$pin"
else bad "after hook failure (docs absent): $(git -C "$C" status --porcelain)"; fi

# --- failure after generate with tracked files under docs/ ---
mkdir -p "$C/docs"
echo "tracked doc" > "$C/docs/x.md"
git -C "$C" add docs/x.md
git -C "$C" commit -qm "tracked docs"
bump v3.3.3
expect "hook failure (docs tracked)" nonzero "failed at step: test hook BUMP_TEST_FAIL_AFTER_GENERATE"
if clean_tree && [ ! -e "$C/docs/resources" ] && [ "$(cat "$C/docs/x.md")" = "tracked doc" ]; then
  ok "restored: tree clean, docs/resources removed, docs/x.md kept"
else bad "after hook failure (docs tracked): $(git -C "$C" status --porcelain)"; fi
unset BUMP_TEST_FAIL_AFTER_GENERATE

# --- preflight refuses local changes under docs/ ---
echo "local edit" >> "$C/docs/x.md"
bump v3.3.3
expect "preflight refuses modified docs/" nonzero "generated paths have local changes"
if [ "$(git -C "$C" status --porcelain)" = " M docs/x.md" ] && grep -qF "local edit" "$C/docs/x.md"; then
  ok "preflight left the local edit untouched"
else bad "preflight changed files: $(git -C "$C" status --porcelain)"; fi
git -C "$C" checkout -- docs/x.md

# --- optional: full successful bump ---
if [ "${BUMP_TEST_SUCCESS:-}" = 1 ]; then
  bump v3.3.3
  expect "successful bump to v3.3.3" 0 "summary in bump-summary.md"
  if [ "$(cat "$C/WAYFINDER_REF")" = v3.3.3 ] && [ -s "$C/bump-summary.md" ] \
    && [ "$(jq -r .wayfinderVersion "$C/schema/wayfinder-manifest.json")" = v3.3.3 ]; then
    ok "WAYFINDER_REF, manifest and bump-summary.md updated"
  else bad "success outputs missing"; fi
  check "bump-summary.md is gitignored" git -C "$C" check-ignore -q bump-summary.md
fi

echo "passed=$pass failed=$failures"
[ "$failures" -eq 0 ]
