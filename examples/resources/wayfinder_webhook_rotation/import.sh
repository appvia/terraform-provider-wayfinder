#!/bin/sh
# Copyright 2026 Appvia Ltd.
# SPDX-License-Identifier: Apache-2.0

set -eu
# <webhook> for a tenant webhook, <workspace>/<webhook> for a workspace webhook.
terraform import wayfinder_webhook_rotation.tenant_hook 'alerts'
terraform import wayfinder_webhook_rotation.alerts 'plat/alerts'
