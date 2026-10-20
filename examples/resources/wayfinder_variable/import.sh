#!/bin/sh
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

set -eu
# [workspace/[environment/[instance/]]]key
terraform import wayfinder_variable.region 'default_region'
terraform import wayfinder_variable.replicas 'plat/dev/replicas'
terraform import wayfinder_variable.instance_replicas 'plat/dev/checkout/replicas'
