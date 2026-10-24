// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package ephemeral_test

import "testing"

func TestWebhookIngestURL_whenTerraformReadsValue(t *testing.T) {
	t.Parallel()

	for _, workspace := range []string{"", "team"} {
		name := "tenant"
		if workspace != "" {
			name = "workspace"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			echoAction(t, "webhook_ingest_url", workspace)
		})
	}
}

func TestWebhookIngestURL_whenTerraformReceivesActionError(t *testing.T) {
	t.Parallel()
	failedAction(t, "webhook_ingest_url")
}

func TestWebhookIngestURL_whenTerraformNameUnknown(t *testing.T) {
	t.Parallel()
	unknownAction(t, "webhook_ingest_url")
}

func TestWebhookIngestURL_whenWriteOnlyConsumerUsed(t *testing.T) {
	writeOnlyAction(t, "webhook_ingest_url")
}
