#!/bin/sh
set -eu
# [workspace/[environment/]]key
terraform import wayfinder_secret.db 'plat/db_credentials'
