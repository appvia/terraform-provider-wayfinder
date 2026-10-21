#!/bin/sh
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

set -eu
# Either the bare feature name or the wayfinder.<name> object name.
terraform import wayfinder_feature_integration.azure azure
terraform import wayfinder_feature_integration.github wayfinder.github
