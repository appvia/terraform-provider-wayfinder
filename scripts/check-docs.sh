#!/usr/bin/env bash
# Fails when a generated resource or data source page has no example, which happens
# when examples/<kind>/<type>/ is missing its resource.tf or data-source.tf.
set -euo pipefail

docs_dir="${1:-docs}"
status=0
for page in "$docs_dir"/resources/*.md "$docs_dir"/data-sources/*.md; do
  if ! grep -q '^## Example Usage$' "$page"; then
    echo "check-docs: $page has no \"## Example Usage\" section" >&2
    status=1
  fi
done
exit "$status"
