#!/usr/bin/env bash
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

# Fails when a generated resource, data source or ephemeral resource page has no
# example, which happens when examples/<kind>/<type>/ is missing its resource.tf,
# data-source.tf or ephemeral-resource.tf.
set -euo pipefail

docs_dir="${1:-docs}"
examples_dir="${2:-examples}"
status=0
shopt -s nullglob
for page in "$docs_dir"/resources/*.md "$docs_dir"/data-sources/*.md "$docs_dir"/ephemeral-resources/*.md; do
  if ! grep -q '^## Example Usage$' "$page"; then
    echo "check-docs: $page has no \"## Example Usage\" section" >&2
    status=1
  fi
done
for page in "$docs_dir"/ephemeral-resources/*.md; do
  example="$examples_dir/ephemeral-resources/wayfinder_$(basename "$page" .md)/ephemeral-resource.tf"
  if [ ! -f "$example" ]; then
    echo "check-docs: $page has no example at $example" >&2
    status=1
  fi
done
exit "$status"
