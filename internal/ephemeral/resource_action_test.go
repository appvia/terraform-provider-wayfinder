// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral_test

import "testing"

func TestResourceAction_whenTerraformReadsValue(t *testing.T) {
	t.Parallel()
	echoAction(t, "resource_action", "")
}

func TestResourceAction_whenTerraformReceivesActionError(t *testing.T) {
	t.Parallel()
	failedAction(t, "resource_action")
}

func TestResourceAction_whenTerraformNameUnknown(t *testing.T) {
	t.Parallel()
	unknownAction(t, "resource_action")
}

func TestResourceAction_whenWriteOnlyConsumerUsed(t *testing.T) {
	writeOnlyAction(t, "resource_action")
}
