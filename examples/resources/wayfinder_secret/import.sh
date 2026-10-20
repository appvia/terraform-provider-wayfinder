#!/bin/sh
set -eu
# [workspace/[environment/[instance/]]]key
terraform import wayfinder_secret.db 'plat/db_credentials'
terraform import wayfinder_secret.checkout_api 'plat/dev/checkout/api_credentials'
