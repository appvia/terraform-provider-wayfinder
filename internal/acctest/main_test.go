// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package acctest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMain runs the sweepers with -sweep=<any label>, otherwise the tests.
func TestMain(m *testing.M) {
	AddSweepers()
	resource.TestMain(m)
}
