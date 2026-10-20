#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Runs a command with its output withheld from the CI log.
#
# Used where a command may quote private Wayfinder source (for example code
# compiled inside a private checkout) and the CI log is public. Stdout and
# stderr go to a temporary file that is deleted on exit. Log contents are
# NEVER printed, not on success and not on failure. Only the label, the exit
# status and a reproduction hint reach the log.
#
# Usage: ci-quiet.sh <label> <command> [args...]
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: ci-quiet.sh <label> <command> [args...]" >&2
  exit 2
fi

label="$1"
shift

log="$(mktemp "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/ci-quiet.XXXXXX")"
trap 'rm -f "$log"' EXIT

echo "==> $label (output withheld: it may quote private Wayfinder source)"

if "$@" >"$log" 2>&1; then
  echo "==> $label: ok"
  exit 0
else
  rc=$?
  echo "::error::$label failed (exit $rc). Output withheld because it may quote private Wayfinder source. Reproduce locally with: $label" >&2
  exit "$rc"
fi
