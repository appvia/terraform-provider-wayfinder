#!/usr/bin/env bash
# Usage: semver-gt.sh A B — exit 0 iff release tag A is strictly newer than B.
# Numeric major/minor/patch comparison; a pre-release sorts below the release
# of the same version; two pre-releases compare lexically. No `sort -V`.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 A B (exit 0 iff A > B)" >&2
  exit 2
fi

awk -v a="$1" -v b="$2" '
function parse(v, out,    core, n, parts) {
  if (v !~ /^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$/) {
    printf "semver-gt: invalid version %s\n", v > "/dev/stderr"
    exit 2
  }
  sub(/^v/, "", v)
  out["pre"] = ""
  n = index(v, "-")
  if (n > 0) { out["pre"] = substr(v, n + 1); v = substr(v, 1, n - 1) }
  split(v, parts, ".")
  out[1] = parts[1] + 0; out[2] = parts[2] + 0; out[3] = parts[3] + 0
}
BEGIN {
  parse(a, A); parse(b, B)
  for (i = 1; i <= 3; i++) {
    if (A[i] > B[i]) exit 0
    if (A[i] < B[i]) exit 1
  }
  if (A["pre"] == B["pre"]) exit 1
  if (A["pre"] == "") exit 0
  if (B["pre"] == "") exit 1
  exit (A["pre"] > B["pre"]) ? 0 : 1
}'
