package generated

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSchemas_whenEmbeddedManifest(t *testing.T) {
	// Given
	ctx := context.Background()
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(Resources()) != len(m.TypedKinds()) || len(DataSources()) != len(m.TypedKinds()) {
		t.Fatal("registry count differs from typed kinds")
	}
	private := regexp.MustCompile(`(?i)varset`)
	for i, factory := range Resources() {
		// When
		r := factory()
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		// Then
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if d := resp.Schema.ValidateImplementation(ctx); d.HasError() {
			t.Fatal(d)
		}
		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "wayfinder"}, &meta)
		if meta.TypeName != ResourceTypeNames()[i] || private.MatchString(meta.TypeName) {
			t.Fatalf("type name %s", meta.TypeName)
		}
		var kind manifest.Kind
		for _, k := range m.TypedKinds() {
			name := "wayfinder_" + strings.TrimPrefix(k.TerraformName, "wayfinder_")
			if k.TerraformName == "" {
				name = "wayfinder_" + convert.SnakeCase(k.Kind)
			}
			if name == meta.TypeName {
				kind = k
				break
			}
		}
		mapping, err := convert.NewMapping(kind)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Schema.Type().TerraformType(ctx).Equal(mapping.TFType()) {
			t.Fatalf("mapping type differs for %s", meta.TypeName)
		}
		checkResourceText(t, resp.Schema.Attributes)
		var ds datasource.SchemaResponse
		DataSources()[i]().Schema(ctx, datasource.SchemaRequest{}, &ds)
		if ds.Diagnostics.HasError() {
			t.Fatal(ds.Diagnostics)
		}
		if d := ds.Schema.ValidateImplementation(ctx); d.HasError() {
			t.Fatal(d)
		}
		checkDataSourceText(t, ds.Schema.Attributes)
		t.Logf("%s: resource/data source valid; mapping equal", meta.TypeName)
	}
}

func checkResourceText(t *testing.T, attrs map[string]rschema.Attribute) {
	t.Helper()
	for name, a := range attrs {
		if strings.Contains(strings.ToLower(name+a.GetDescription()+a.GetMarkdownDescription()), "varset") {
			t.Fatal("private terminology", name)
		}
		switch v := a.(type) {
		case rschema.SingleNestedAttribute:
			checkResourceText(t, v.Attributes)
		case rschema.ListNestedAttribute:
			checkResourceText(t, v.NestedObject.Attributes)
		case rschema.MapNestedAttribute:
			checkResourceText(t, v.NestedObject.Attributes)
		}
	}
}

func checkDataSourceText(t *testing.T, attrs map[string]dschema.Attribute) {
	t.Helper()
	for name, a := range attrs {
		if strings.HasSuffix(name, "_wo") || strings.HasSuffix(name, "_wo_version") {
			t.Fatal("secret in data source", name)
		}
		if strings.Contains(strings.ToLower(name+a.GetDescription()+a.GetMarkdownDescription()), "varset") {
			t.Fatal("private terminology", name)
		}
		switch v := a.(type) {
		case dschema.SingleNestedAttribute:
			checkDataSourceText(t, v.Attributes)
		case dschema.ListNestedAttribute:
			checkDataSourceText(t, v.NestedObject.Attributes)
		case dschema.MapNestedAttribute:
			checkDataSourceText(t, v.NestedObject.Attributes)
		}
	}
}

func TestRegistry_whenEmbeddedManifest_hasOneResourceAndDataSourcePerTypedKind(t *testing.T) {
	// Given the manifest compiled into the provider.
	m, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	// When / Then every typed kind has exactly one resource and one data source.
	typed := len(m.TypedKinds())
	if len(Resources()) != typed || len(ResourceTypeNames()) != typed {
		t.Fatalf("resources = %d, type names = %d, typed kinds = %d", len(Resources()), len(ResourceTypeNames()), typed)
	}
	if len(DataSources()) != typed {
		t.Fatalf("data sources = %d, typed kinds = %d", len(DataSources()), typed)
	}
	t.Logf("typed kinds = resources = data sources = %d", typed)
}

func TestVersion_whenVersionedKubeResourcePlan(t *testing.T) {
	// Given
	a := ResourceSchema_kube_resource_plan(context.Background()).Attributes["version"].(rschema.StringAttribute)
	// When / Then
	if !a.Required || len(a.PlanModifiers) == 0 || len(a.Validators) == 0 {
		t.Fatal("version must be required, replacing, validated")
	}
	for _, value := range []string{"1.0.0", "1.2.3-rc.1+build", "v1.0.0", "01.0.0", "1.0.0-01"} {
		var resp validator.StringResponse
		a.Validators[0].ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, &resp)
		valid := value == "1.0.0" || value == "1.2.3-rc.1+build"
		if resp.Diagnostics.HasError() == valid {
			t.Errorf("semver validation %q", value)
		}
	}
}

func TestWriteOnly_whenExternalIdentityNestedSecrets(t *testing.T) {
	// Given
	s := ResourceSchema_external_identity(context.Background())
	// When
	bearer := s.Attributes["bearer_token"].(rschema.SingleNestedAttribute)
	a := bearer.Attributes["token_wo"].(rschema.StringAttribute)
	v := bearer.Attributes["token_wo_version"].(rschema.StringAttribute)
	headers := s.Attributes["api_key"].(rschema.SingleNestedAttribute).Attributes["headers_wo"].(rschema.MapAttribute)
	ds := DataSourceSchema_external_identity(context.Background()).Attributes["bearer_token"].(dschema.SingleNestedAttribute)
	// Then
	if !a.WriteOnly || !a.Sensitive || !a.Required || a.Computed || len(a.PlanModifiers) != 0 || !v.Optional || v.WriteOnly {
		t.Fatal("invalid write-only pair")
	}
	if !headers.WriteOnly || !headers.Sensitive {
		t.Fatal("sensitive map must be write-only")
	}
	if _, ok := ds.Attributes["token_wo"]; ok {
		t.Fatal("data source secret")
	}
	if _, ok := ds.Attributes["token_wo_version"]; ok {
		t.Fatal("data source companion")
	}
}

func TestScopes_whenKindScopesDiffer(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name                   string
		schema                 rschema.Schema
		workspace, environment string
	}{
		{"tenant only", ResourceSchema_workspace(ctx), "", ""},
		{"workspace only", ResourceSchema_environment(ctx), "required", ""},
		{"environment only", ResourceSchema_cloud_resource(ctx), "required", "required"},
		{"tenant or workspace", ResourceSchema_integration(ctx), "optional", ""},
		{"any scope", ResourceSchema_external_identity(ctx), "optional", "optional"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given / When / Then
			for attr, want := range map[string]string{"workspace": tc.workspace, "environment": tc.environment} {
				if got := presence(tc.schema, attr); got != want {
					t.Fatalf("%s = %q, want %q", attr, got, want)
				}
			}
		})
	}
}

func presence(s rschema.Schema, name string) string {
	a, ok := s.Attributes[name].(rschema.StringAttribute)
	switch {
	case !ok:
		return ""
	case a.Required:
		return "required"
	default:
		return "optional"
	}
}
