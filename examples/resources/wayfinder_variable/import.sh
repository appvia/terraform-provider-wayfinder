#!/bin/sh
set -eu
# [workspace/[environment/]]key
terraform import wayfinder_variable.region 'default_region'
terraform import wayfinder_variable.replicas 'plat/dev/replicas'
