// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package featureintegration_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appvia/terraform-provider-wayfinder/internal/featureintegration"
)

func expectWrites(e env, post, put int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		for method, want := range map[string]int{
			http.MethodPost: post, http.MethodPut: put, http.MethodDelete: 0,
		} {
			if got := e.fake.CountRequests(method, ""); got != want {
				return fmt.Errorf("%s requests = %d, want %d; log: %+v", method, got, want, e.fake.Requests())
			}
		}

		return nil
	}
}

func (e env) changeFeature(t *testing.T, name string, change func(map[string]any)) {
	t.Helper()

	ref := featureintegration.IntegrationRef(name)

	obj, err := e.client.Get(t.Context(), ref)
	if err != nil {
		t.Fatal(err)
	}

	change(obj["spec"].(map[string]any))

	if _, err := e.client.Update(t.Context(), ref, obj); err != nil {
		t.Fatal(err)
	}
}

func lifecycleResource(t *testing.T, e env, name string) (frameworkresource.Resource, tfsdk.State) {
	t.Helper()

	r := featureintegration.NewResource()

	var configured frameworkresource.ConfigureResponse
	r.(frameworkresource.ResourceWithConfigure).Configure(t.Context(),
		frameworkresource.ConfigureRequest{ProviderData: e.client}, &configured)

	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}

	var schema frameworkresource.SchemaResponse
	r.Schema(t.Context(), frameworkresource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	value := types.ObjectValueMust(schema.Schema.Type().(types.ObjectType).AttrTypes, map[string]attr.Value{
		"id": types.StringValue("wayfinder." + name), "name": types.StringValue(name),
		"params":                  types.MapNull(jsontypes.NormalizedType{}),
		"keep_enabled_on_destroy": types.BoolValue(false),
		"enabled":                 types.BoolValue(true), "type": types.StringValue("feature"),
	})

	if diags := state.Set(t.Context(), value); diags.HasError() {
		t.Fatal(diags)
	}

	return r, state
}
