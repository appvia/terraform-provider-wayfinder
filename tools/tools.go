// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// Package tools pins developer tooling dependencies.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
