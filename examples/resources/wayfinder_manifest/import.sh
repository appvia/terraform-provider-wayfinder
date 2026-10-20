#!/bin/sh
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

set -eu
# <group>/<Kind>/[workspace/[environment/]]name[@version]
terraform import wayfinder_manifest.developers 'org/Group/developers'
