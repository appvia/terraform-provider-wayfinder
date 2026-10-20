// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package vars_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/appvia/terraform-provider-wayfinder/internal/vars"
)

// TestSchemas_never_mention_the_internal_object_name keeps user-facing text free of the
// internal storage object name (plan R13).
func TestSchemas_never_mention_the_internal_object_name(t *testing.T) {
	ctx := context.Background()
	var texts []string
	for _, f := range vars.Resources() {
		var resp resource.SchemaResponse
		f().Schema(ctx, resource.SchemaRequest{}, &resp)
		texts = append(texts, resp.Schema.Description, resp.Schema.MarkdownDescription)
		for name, a := range resp.Schema.Attributes {
			texts = append(texts, name, a.GetDescription(), a.GetMarkdownDescription())
		}
	}
	for _, f := range vars.DataSources() {
		var resp datasource.SchemaResponse
		f().Schema(ctx, datasource.SchemaRequest{}, &resp)
		texts = append(texts, resp.Schema.Description, resp.Schema.MarkdownDescription)
		for name, a := range resp.Schema.Attributes {
			texts = append(texts, name, a.GetDescription(), a.GetMarkdownDescription())
		}
	}
	if len(texts) < 10 {
		t.Fatalf("collected only %d texts", len(texts))
	}
	for _, s := range texts {
		if strings.Contains(strings.ToLower(s), "varset") {
			t.Errorf("user-facing text mentions the internal object: %q", s)
		}
	}
}
