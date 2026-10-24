// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package webhookrotation

import "testing"

var SegmentValidator = segmentValidator{}

// SetNewRotationID swaps the rotation ID source until t ends. Tests using it must not be parallel.
func SetNewRotationID(t *testing.T, fn func() (string, error)) {
	previous := newRotationID
	newRotationID = fn

	t.Cleanup(func() { newRotationID = previous })
}
