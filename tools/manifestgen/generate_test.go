//go:build wfmanifestgen

package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/appvia/wayfinder/hack/tfprovider-manifestgen/manifest"
	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
	"github.com/stretchr/testify/require"
)

type fixtureObject struct {
	corev3.Object `json:"-"`
	Spec          fixtureSpec `json:"spec"`
}

func (*fixtureObject) GroupVersionKind() corev3.GroupVersionKind {
	return corev3.GroupVersionKind{Group: "fixtures", Version: "v3", Kind: "ExternalIdentity"}
}
func (*fixtureObject) APIPath() string { return "/identities" }
func (*fixtureObject) SupportedScopes() corev3.ObjectScope {
	return corev3.ObjectScopeTenant | corev3.ObjectScopeSystem
}

type fixtureSpec struct {
	Token       string            `json:"token,omitempty"`   // +encrypted, +required
	Headers     map[string]string `json:"headers,omitempty"` // +encrypted
	Description string            `json:"description"`
	Node        *fixtureNode      `json:"node,omitempty"`
	Rules       []fixtureRule     `json:"rules,omitempty"`
	Raw         json.RawMessage   `json:"raw,omitempty"`
}
type fixtureNode struct {
	Next  *fixtureNode `json:"next,omitempty"`
	Label string       `json:"label"`
}
type fixtureRule struct {
	Name string `json:"name"`
}

func (fixtureSpec) SwaggerValidation() map[string]corev3.FieldValidation {
	return map[string]corev3.FieldValidation{"token": {Required: true}}
}
func (fixtureSpec) EncryptedFields() map[string]bool {
	return map[string]bool{"token": true, "headers": true}
}

func fixtureGenerator() (generator, KindsFile) {
	return generator{registry: registeredKinds([]corev3.Object{&fixtureObject{}}),
			version: "v-fixture", commit: strings.Repeat("a", 40)},
		KindsFile{Kinds: []KindEntry{{Group: "fixtures", Kind: "ExternalIdentity", Class: ClassTyped}}}
}

func TestGenerate_marks_encrypted_fields_sensitive(t *testing.T) {
	// Given a registered fixture exposing generated +encrypted metadata.
	g, kinds := fixtureGenerator()
	// When generated.
	m, err := g.generate(kinds, FieldsFile{})
	// Then both scalar and map sensitivity are on the field itself.
	require.NoError(t, err)
	require.True(t, m.Kinds[0].Spec.Properties["token"].Sensitive)
	require.True(t, m.Kinds[0].Spec.Properties["headers"].Sensitive)
	require.False(t, m.Kinds[0].Spec.Properties["headers"].Values.Sensitive)
}

func TestGenerate_required_uses_validation_not_json_tag(t *testing.T) {
	// Given an omitempty required field and a non-omitempty optional field.
	g, kinds := fixtureGenerator()
	// When generated.
	m, err := g.generate(kinds, FieldsFile{})
	// Then only validation controls Required.
	require.NoError(t, err)
	require.True(t, m.Kinds[0].Spec.Properties["token"].Required)
	require.False(t, m.Kinds[0].Spec.Properties["description"].Required)
}

func TestGenerate_terminates_when_spec_is_recursive(t *testing.T) {
	// Given an ordinary object whose spec contains a recursive node.
	g, kinds := fixtureGenerator()
	// When generated.
	m, err := g.generate(kinds, FieldsFile{})
	// Then the cycle is JSON without discarding siblings.
	require.NoError(t, err)
	node := m.Kinds[0].Spec.Properties["node"]
	require.Equal(t, "json", node.Properties["next"].Type)
	require.Equal(t, "string", node.Properties["label"].Type)
	require.Equal(t, "json", m.Kinds[0].Spec.Properties["raw"].Type)
}

func TestGenerate_omits_ignored_field(t *testing.T) {
	// Given parsed overrides targeting an array item's field.
	g, kinds := fixtureGenerator()
	fields, err := ParseFields([]byte("fields:\n  - {group: fixtures, kind: ExternalIdentity, path: spec.rules.name, ignore: true, reason: owned}\n"))
	require.NoError(t, err)
	// When generated.
	m, err := g.generate(kinds, fields)
	// Then the ignored property and its order entry disappear.
	require.NoError(t, err)
	require.Empty(t, m.Kinds[0].Spec.Properties["rules"].Items.Properties)
	require.Empty(t, m.Kinds[0].Spec.Properties["rules"].Items.PropertyOrder)
}

func TestGenerate_names_entry_and_tag_when_path_missing(t *testing.T) {
	// Given a stale override.
	g, kinds := fixtureGenerator()
	fields, err := ParseFields([]byte("fields:\n  - {group: fixtures, kind: ExternalIdentity, path: spec.doesNotExist, computed: true}\n"))
	require.NoError(t, err)
	// When generated.
	_, err = g.generate(kinds, fields)
	// Then the failure identifies entry, path and pin.
	require.ErrorContains(t, err, "fields[0] fixtures/ExternalIdentity spec.doesNotExist")
	require.ErrorContains(t, err, "v-fixture")
}

func TestGenerate_is_deterministic_and_valid(t *testing.T) {
	// Given a fixture registry.
	g, kinds := fixtureGenerator()
	var outputs [][]byte
	// When independently generated twice.
	for range 2 {
		m, err := g.generate(kinds, FieldsFile{})
		require.NoError(t, err)
		require.NoError(t, m.Validate())
		data, err := manifest.Encode(m)
		require.NoError(t, err)
		outputs = append(outputs, data)
	}
	// Then bytes match and property order follows Go declaration order.
	require.Equal(t, outputs[0], outputs[1])
	m, err := manifest.Parse(outputs[0])
	require.NoError(t, err)
	require.Equal(t, []string{"token", "headers", "description", "node", "rules", "raw"}, m.Kinds[0].Spec.PropertyOrder)
}

func TestControllerInventory_contains_real_controllers(t *testing.T) {
	// Given the copied generator location inside the checkout.
	root, err := checkoutRoot()
	require.NoError(t, err)
	// When AST inventory is collected.
	inventory, err := controllerInventory(root)
	// Then known controllers cannot silently vanish.
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(inventory), 20)
	require.True(t, inventory["WorkloadIdentity"])
}

func TestGenerate_real_registry_passes_contract(t *testing.T) {
	// Given real overrides and controller inventory.
	kinds, err := LoadKinds(os.Getenv("MANIFESTGEN_KINDS"))
	require.NoError(t, err)
	fields, err := LoadFields(os.Getenv("MANIFESTGEN_FIELDS"))
	require.NoError(t, err)
	root, err := checkoutRoot()
	require.NoError(t, err)
	inventory, err := controllerInventory(root)
	require.NoError(t, err)
	g := generator{registry: wayfinderRegistry(), controllers: inventory, version: "v3.3.2"}
	// When generated.
	m, err := g.generate(kinds, fields)
	// Then actual API types meet the wire contract.
	require.NoError(t, err)
	require.NoError(t, m.Validate())
}

func TestReflectSpec_rejects_snake_collision(t *testing.T) {
	// Given two distinct JSON names with the same Terraform spelling.
	type collision struct {
		First  string `json:"fooBar"`
		Second string `json:"foo_bar"`
	}
	// When reflected.
	_, err := reflectField(reflect.TypeFor[collision](), make(map[reflect.Type]bool))
	// Then the names are reported rather than silently merged.
	require.ErrorContains(t, err, "fooBar")
	require.ErrorContains(t, err, "foo_bar")
}

func TestSnake_keepsPluralAcronymsWhole(t *testing.T) {
	// Given / When / Then the spelling matches the provider's convert.SnakeCase.
	require.Equal(t, "iam_policy_arns", snake("iamPolicyARNs"))
	require.Equal(t, "private_zone_vpcs", snake("privateZoneVPCs"))
	require.Equal(t, "urls_to_check", snake("URLsToCheck"))
	require.Equal(t, "http_server_url", snake("HTTPServerURL"))
}
