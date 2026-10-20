// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package generated

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"sigs.k8s.io/yaml"
)

// fieldEntry mirrors tools/manifestgen/overrides.go FieldEntry; that package is
// build-tagged and depends on the private module, so it cannot be imported.
type fieldEntry struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Immutable bool   `json:"immutable,omitempty"`
	Computed  bool   `json:"computed,omitempty"`
	Ignore    bool   `json:"ignore,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type pin struct {
	group, kind, path           string
	immutable, computed, ignore bool
}

// pinned is tools/manifestgen/fields.yaml in file order, reasons excluded.
// Changing an override changes provider behaviour; update this deliberately.
var pinned = []pin{
	{"cloudaccess", "AzureSubscription", "spec.tenantID", true, false, false},
	{"cloudaccess", "GCPProject", "spec.projectNumber", true, false, false},
	{"inference", "AIAgent", "spec.inputSchema", false, false, true},
	{"inference", "AIContext", "spec.use", false, true, false},
	{"integrations", "ExternalIdentity", "spec.aws.account", true, false, false},
	{"integrations", "ExternalIdentity", "spec.azure.subscription", true, false, false},
	{"integrations", "ExternalIdentity", "spec.azure.tenantID", false, true, false},
	{"integrations", "ExternalIdentity", "spec.cloudAccountRef", false, true, false},
	{"integrations", "ExternalIdentity", "spec.gcp.project", true, false, false},
	{"integrations", "ExternalIdentity", "spec.gcp.projectNumber", false, true, false},
	{"integrations", "ExternalIdentity", "spec.type", true, false, false},
	{"integrations", "ExternalIdentity", "spec.usableBy", false, true, false},
	{"integrations", "Integration", "spec.templateRef", true, true, false},
	{"integrations", "Integration", "spec.type", true, true, false},
	{"integrations", "MCP", "spec.definitionRef", false, true, false},
	{"integrations", "WebAPI", "spec.definitionRef", false, true, false},
	{"integrations", "Webhook", "spec.definitionRef", false, true, false},
	{"integrations", "Webhook", "spec.ingestToken", false, false, true},
	{"integrations", "Webhook", "spec.ingestTokenHash", false, false, true},
	{"kuberesources", "KubeResource", "spec.workloadIdentity.cloudResourceName", true, false, false},
	{"kuberesources", "KubeResourcePlan", "spec.workloadIdentity.cloudResourceName", true, false, false},
	{"kubernetes", "Cluster", "spec.cloudResourceName", true, true, false},
	{"kubernetes", "ClusterRoleBinding", "spec.clusterRef", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.clusterRoleName", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.expiration", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.namespace", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.roleBindingType", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.subject", true, false, false},
	{"kubernetes", "ClusterRoleBinding", "spec.subject.kind", false, true, false},
	{"kubernetes", "ClusterRoleBinding", "spec.subject.name", false, true, false},
	{"orchestration", "Workflow", "spec.enabled", false, true, false},
	{"org", "Environment", "spec.slug", true, true, false},
	{"org", "Invite", "spec.autoJoin", false, true, false},
	{"org", "Invite", "spec.expiresAt", false, true, false},
	{"org", "Invite", "spec.scope", false, true, false},
	{"org", "Invite", "spec.tokenID", false, true, false},
	{"org", "ServiceAccountCredential", "spec.clientID", false, true, false},
	{"org", "ServiceAccountCredential", "spec.federated.audiences", false, true, false},
	{"org", "ServiceAccountCredential", "spec.method", true, false, false},
	{"org", "ServiceAccountCredential", "spec.tokenHash", false, false, true},
	{"org", "Workspace", "spec.slug", true, true, false},
	{"org", "Workspace", "spec.type", false, true, false},
	{"sourcecontrol", "GitHubOrg", "spec.installationID", false, false, true},
	{"sourcecontrol", "GitHubOrg", "spec.installationRequestID", false, false, true},
	{"sourcecontrol", "GitHubOrg", "spec.organization", true, false, false},
	{"stacks", "Stack", "spec.createdFrom", true, false, false},
}

// pinDiff describes every position where got differs from want; empty means equal.
func pinDiff(want, got []pin) string {
	var diffs []string

	for i := range max(len(want), len(got)) {
		switch {
		case i >= len(got):
			diffs = append(diffs, fmt.Sprintf("[%d] missing %+v", i, want[i]))
		case i >= len(want):
			diffs = append(diffs, fmt.Sprintf("[%d] unexpected %+v", i, got[i]))
		case want[i] != got[i]:
			diffs = append(diffs, fmt.Sprintf("[%d] want %+v, got %+v", i, want[i], got[i]))
		}
	}

	return strings.Join(diffs, "; ")
}

func loadFieldsYAML(t *testing.T) []pin {
	t.Helper()

	data, err := os.ReadFile("../../tools/manifestgen/fields.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var file struct {
		Fields []fieldEntry `json:"fields"`
	}
	if err := yaml.UnmarshalStrict(data, &file); err != nil {
		t.Fatal(err)
	}

	out := make([]pin, 0, len(file.Fields))
	for _, e := range file.Fields {
		out = append(out, pin{e.Group, e.Kind, e.Path, e.Immutable, e.Computed, e.Ignore})
	}

	return out
}

// specField walks spec along a dotted path, descending through array items and
// map values like tools/manifestgen applyOverrides. It returns the final
// property's parent and whether the final property exists.
func specField(spec *manifest.Field, path string) (*manifest.Field, bool) {
	parent := spec

	parts := strings.Split(path, ".")[1:]
	for j, part := range parts {
		for parent != nil && (parent.Type == "array" || parent.Type == "map") {
			if parent.Type == "array" {
				parent = parent.Items
			} else {
				parent = parent.Values
			}
		}

		if parent == nil {
			return nil, false
		}

		field := parent.Properties[part]
		if field == nil || j == len(parts)-1 {
			return field, field != nil
		}

		parent = field
	}

	return nil, false
}

func resourceSchemaFor(ctx context.Context, t *testing.T, k manifest.Kind) rschema.Schema {
	t.Helper()

	name := "wayfinder_" + strings.TrimPrefix(k.TerraformName, "wayfinder_")
	if k.TerraformName == "" {
		name = "wayfinder_" + convert.SnakeCase(k.Kind)
	}

	for _, factory := range Resources() {
		r := factory()

		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "wayfinder"}, &meta)

		if meta.TypeName != name {
			continue
		}

		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		return resp.Schema
	}

	t.Fatalf("no generated resource %s", name)

	return rschema.Schema{}
}

// modifierDescriptions returns the descriptions of every plan modifier on attr.
func modifierDescriptions(ctx context.Context, attr any) []string {
	var out []string

	add := func(n int, desc func(int) string) {
		for i := range n {
			out = append(out, desc(i))
		}
	}

	if a, ok := attr.(interface{ StringPlanModifiers() []planmodifier.String }); ok {
		m := a.StringPlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ BoolPlanModifiers() []planmodifier.Bool }); ok {
		m := a.BoolPlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ Int64PlanModifiers() []planmodifier.Int64 }); ok {
		m := a.Int64PlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ Float64PlanModifiers() []planmodifier.Float64 }); ok {
		m := a.Float64PlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ ListPlanModifiers() []planmodifier.List }); ok {
		m := a.ListPlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ MapPlanModifiers() []planmodifier.Map }); ok {
		m := a.MapPlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	if a, ok := attr.(interface{ ObjectPlanModifiers() []planmodifier.Object }); ok {
		m := a.ObjectPlanModifiers()
		add(len(m), func(i int) string { return m[i].Description(ctx) })
	}

	return out
}

func TestFieldOverrides_whenFieldsYAMLPinned(t *testing.T) {
	t.Parallel()
	// Given
	ctx := context.Background()
	requiresReplace := stringplanmodifier.RequiresReplace().Description(ctx)

	useState := stringplanmodifier.UseStateForUnknown().Description(ctx)
	if objectplanmodifier.RequiresReplace().Description(ctx) != requiresReplace ||
		listplanmodifier.UseStateForUnknown().Description(ctx) != useState ||
		objectplanmodifier.UseStateForUnknown().Description(ctx) != useState {
		t.Fatal("framework plan modifier descriptions differ across attribute types; update modifier detection")
	}

	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	// When
	entries := loadFieldsYAML(t)
	// Then
	if d := pinDiff(pinned, entries); d != "" {
		t.Fatalf("fields.yaml changed: %s; update pinned in fields_pin_test.go deliberately", d)
	}

	for _, e := range entries {
		k, ok := m.FindKind(e.group, e.kind)
		if !ok {
			t.Fatalf("%s/%s not in embedded manifest", e.group, e.kind)
		}

		field, ok := specField(k.Spec, e.path)
		if e.ignore {
			if ok {
				t.Fatalf("%s/%s %s: ignored field present in manifest", e.group, e.kind, e.path)
			}
		} else if !ok || field.Immutable != e.immutable || field.Computed != e.computed {
			t.Fatalf("%s/%s %s: manifest field %+v does not match override", e.group, e.kind, e.path, field)
		}

		if !k.Typed {
			continue
		}

		mapping, err := convert.NewMapping(*k)
		if err != nil {
			t.Fatal(err)
		}

		p, ok := mapping.AttributePath(e.path)
		if e.ignore {
			if ok {
				t.Fatalf("%s/%s %s: ignored field maps to %s", e.group, e.kind, e.path, p)
			}

			continue
		}

		if !ok {
			t.Fatalf("cannot resolve %s/%s %s to a Terraform attribute", e.group, e.kind, e.path)
		}

		attr, diags := resourceSchemaFor(ctx, t, *k).AttributeAtPath(ctx, p)
		if diags.HasError() {
			t.Fatal(diags)
		}

		mods := modifierDescriptions(ctx, attr)
		if e.computed && (!attr.IsComputed() || !slices.Contains(mods, useState)) {
			t.Errorf("%s/%s %s: want computed with UseStateForUnknown, got computed=%v modifiers=%q", e.group, e.kind, e.path, attr.IsComputed(), mods)
		}

		if e.immutable != slices.Contains(mods, requiresReplace) {
			t.Errorf("%s/%s %s: immutable=%v but modifiers=%q", e.group, e.kind, e.path, e.immutable, mods)
		}
	}

	t.Logf("checked %d fields.yaml entries against manifest and schema", len(entries))
}

func TestFieldOverrides_whenPinMismatches(t *testing.T) {
	t.Parallel()
	// Given
	flipped := slices.Clone(pinned)
	flipped[0].immutable = !flipped[0].immutable

	cases := map[string][]pin{
		"flag flipped":  flipped,
		"entry dropped": pinned[1:],
		"entry added":   append(slices.Clone(pinned), pin{"org", "Workspace", "spec.extra", true, false, false}),
	}
	for name, got := range cases {
		// When
		d := pinDiff(pinned, got)
		// Then
		if d == "" {
			t.Errorf("%s: want non-empty diff", name)
		}
	}
}
