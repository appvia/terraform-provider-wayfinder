#!/usr/bin/env bash
# Prints the newest STABLE Wayfinder release tag (vX.Y.Z, no suffix) at
# ${WAYFINDER_GIT_URL}. Token transport: see wayfinder-checkout.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/wayfinder-git.sh
. "$ROOT/scripts/wayfinder-git.sh"

tags="$(wf_git ls-remote --tags --refs "$WF_URL" 'v*' | awk '{ sub(/^refs\/tags\//, "", $2); print $2 }' \
  | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true)"
if [ -z "$tags" ]; then
  echo "no stable vX.Y.Z tags found in $(wf_redacted_url)" >&2
  exit 1
fi

printf '%s\n' "$tags" | awk '
{
  v = substr($0, 2); split(v, p, ".")
  if (best == "" || p[1] + 0 > b1 || (p[1] + 0 == b1 && (p[2] + 0 > b2 || (p[2] + 0 == b2 && p[3] + 0 > b3)))) {
    best = $0; b1 = p[1] + 0; b2 = p[2] + 0; b3 = p[3] + 0
  }
}
END { print best }'
