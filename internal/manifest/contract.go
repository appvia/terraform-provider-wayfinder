// Package manifest defines the wire contract between the Wayfinder API
// manifest generator and the Terraform provider.
//
// This file is copied verbatim into the Wayfinder checkout by the generator,
// so it MUST import only the Go standard library.
package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

// ManifestVersion is the only manifest schema version this contract understands.
const ManifestVersion = 1

// Known scopes a kind may be available in.
var knownScopes = map[string]struct{}{
	"tenant":      {},
	"workspace":   {},
	"environment": {},
}

// Known Field.Type values.
var knownTypes = map[string]struct{}{
	"string":  {},
	"boolean": {},
	"integer": {},
	"number":  {},
	"object":  {},
	"array":   {},
	"map":     {},
	"json":    {},
}

var (
	kindNameRE      = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	terraformNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	propertyNameRE  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	commitRE        = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Manifest is the document produced by the generator.
type Manifest struct {
	ManifestVersion  int      `json:"manifestVersion"`
	WayfinderVersion string   `json:"wayfinderVersion"`
	WayfinderCommit  string   `json:"wayfinderCommit,omitempty"`
	Kinds            []Kind   `json:"kinds"`
	Excluded         []string `json:"excluded,omitempty"`
}

// Kind describes one Wayfinder object kind. Spec describes the spec subtree only
// (never metadata or status).
type Kind struct {
	Kind          string   `json:"kind"`
	Group         string   `json:"group"`
	Version       string   `json:"version"`
	APIPath       string   `json:"apiPath"`
	Typed         bool     `json:"typed"`
	Scopes        []string `json:"scopes"`
	Versioned     bool     `json:"versioned"`
	HasStatus     bool     `json:"hasStatus"`
	WaitForReady  bool     `json:"waitForReady"`
	TerraformName string   `json:"terraformName,omitempty"`
	Description   string   `json:"description,omitempty"`
	Spec          *Field   `json:"spec,omitempty"`
}

// Field describes one schema node. Properties are keyed by JSON name; their
// document order is carried in PropertyOrder (see MarshalJSON).
type Field struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Computed    bool              `json:"computed,omitempty"`
	Immutable   bool              `json:"immutable,omitempty"`
	Sensitive   bool              `json:"sensitive,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
	EnumDocs    []EnumDoc         `json:"enumDocs,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	MinLength   *int              `json:"minLength,omitempty"`
	MaxLength   *int              `json:"maxLength,omitempty"`
	Minimum     *float64          `json:"minimum,omitempty"`
	Maximum     *float64          `json:"maximum,omitempty"`
	Format      string            `json:"format,omitempty"`
	Properties  map[string]*Field `json:"-"`
	// PropertyOrder is not serialized. Decoding sets it to document order;
	// encoding emits PropertyOrder first and then any remaining names sorted.
	PropertyOrder []string `json:"-"`
	Items         *Field   `json:"items,omitempty"`
	Values        *Field   `json:"values,omitempty"`
}

// EnumDoc documents one enum value.
type EnumDoc struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// fieldWire is the on-the-wire shape of Field. Declaration order is output order.
type fieldWire struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Computed    bool            `json:"computed,omitempty"`
	Immutable   bool            `json:"immutable,omitempty"`
	Sensitive   bool            `json:"sensitive,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	EnumDocs    []EnumDoc       `json:"enumDocs,omitempty"`
	Pattern     string          `json:"pattern,omitempty"`
	MinLength   *int            `json:"minLength,omitempty"`
	MaxLength   *int            `json:"maxLength,omitempty"`
	Minimum     *float64        `json:"minimum,omitempty"`
	Maximum     *float64        `json:"maximum,omitempty"`
	Format      string          `json:"format,omitempty"`
	Properties  json.RawMessage `json:"properties,omitempty"`
	Items       *Field          `json:"items,omitempty"`
	Values      *Field          `json:"values,omitempty"`
}

// MarshalJSON emits properties in PropertyOrder, then any unlisted names sorted.
func (f Field) MarshalJSON() ([]byte, error) {
	w := fieldWire{
		Type: f.Type, Description: f.Description, Required: f.Required,
		Computed: f.Computed, Immutable: f.Immutable, Sensitive: f.Sensitive,
		Enum: f.Enum, EnumDocs: f.EnumDocs, Pattern: f.Pattern,
		MinLength: f.MinLength, MaxLength: f.MaxLength,
		Minimum: f.Minimum, Maximum: f.Maximum, Format: f.Format,
		Items: f.Items, Values: f.Values,
	}
	if len(f.Properties) > 0 {
		raw, err := marshalProperties(f.Properties, f.PropertyOrder)
		if err != nil {
			return nil, err
		}
		w.Properties = raw
	}
	return json.Marshal(w)
}

// UnmarshalJSON decodes a Field strictly (unknown keys are errors) and records
// property names in document order.
func (f *Field) UnmarshalJSON(data []byte) error {
	var w fieldWire
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return err
	}
	*f = Field{
		Type: w.Type, Description: w.Description, Required: w.Required,
		Computed: w.Computed, Immutable: w.Immutable, Sensitive: w.Sensitive,
		Enum: w.Enum, EnumDocs: w.EnumDocs, Pattern: w.Pattern,
		MinLength: w.MinLength, MaxLength: w.MaxLength,
		Minimum: w.Minimum, Maximum: w.Maximum, Format: w.Format,
		Items: w.Items, Values: w.Values,
	}
	if len(w.Properties) == 0 || string(bytes.TrimSpace(w.Properties)) == "null" {
		return nil
	}
	props, order, err := decodeOrderedProperties(w.Properties)
	if err != nil {
		return err
	}
	f.Properties = props
	f.PropertyOrder = order
	return nil
}

func marshalProperties(props map[string]*Field, order []string) (json.RawMessage, error) {
	names := orderedNames(props, order)
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			buf.WriteByte(',')
		}
		child := props[name]
		if child == nil {
			return nil, fmt.Errorf("property %q is nil", name)
		}
		key, err := json.Marshal(name)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(child)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// orderedNames returns names listed in order (that exist in props, first
// occurrence wins), followed by the remaining names in sorted order.
func orderedNames(props map[string]*Field, order []string) []string {
	seen := make(map[string]struct{}, len(props))
	names := make([]string, 0, len(props))
	for _, name := range order {
		if _, ok := props[name]; !ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	rest := make([]string, 0, len(props)-len(names))
	for name := range props {
		if _, ok := seen[name]; !ok {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	return append(names, rest...)
}

func decodeOrderedProperties(raw json.RawMessage) (map[string]*Field, []string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, nil, fmt.Errorf("properties: expected object")
	}
	props := map[string]*Field{}
	var order []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		name, ok := tok.(string)
		if !ok {
			return nil, nil, fmt.Errorf("properties: expected string key")
		}
		if _, dup := props[name]; dup {
			return nil, nil, fmt.Errorf("properties: duplicate key %q", name)
		}
		var child Field
		if err := dec.Decode(&child); err != nil {
			return nil, nil, fmt.Errorf("properties.%s: %w", name, err)
		}
		props[name] = &child
		order = append(order, name)
	}
	return props, order, nil
}

// Decode strictly decodes a manifest document without validating it.
func Decode(data []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Encode renders the manifest as 2-space indented JSON with a trailing newline.
func Encode(m *Manifest) ([]byte, error) {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Parse decodes and validates a manifest document.
func Parse(data []byte) (*Manifest, error) {
	m, err := Decode(data)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// Validate checks the manifest. The first violation is returned; its message
// starts with the JSON path of the offending value.
func (m *Manifest) Validate() error {
	if m.ManifestVersion != ManifestVersion {
		return fmt.Errorf("manifestVersion: must be %d, got %d", ManifestVersion, m.ManifestVersion)
	}
	if m.WayfinderVersion == "" {
		return fmt.Errorf("wayfinderVersion: required")
	}
	if m.WayfinderCommit != "" && !commitRE.MatchString(m.WayfinderCommit) {
		return fmt.Errorf("wayfinderCommit: must be 40 lowercase hex characters or empty, got %q", m.WayfinderCommit)
	}
	for i := range m.Kinds {
		if err := m.Kinds[i].validate(fmt.Sprintf("kinds[%d]", i)); err != nil {
			return err
		}
		if i > 0 && m.Kinds[i-1].Kind >= m.Kinds[i].Kind {
			return fmt.Errorf("kinds[%d].kind: kinds must be sorted by kind name and unique, got %q after %q",
				i, m.Kinds[i].Kind, m.Kinds[i-1].Kind)
		}
	}
	kindNames := make(map[string]struct{}, len(m.Kinds))
	for _, k := range m.Kinds {
		kindNames[k.Kind] = struct{}{}
	}
	for i, name := range m.Excluded {
		if i > 0 && m.Excluded[i-1] >= name {
			return fmt.Errorf("excluded[%d]: must be sorted and unique, got %q after %q", i, name, m.Excluded[i-1])
		}
		if _, ok := kindNames[name]; ok {
			return fmt.Errorf("excluded[%d]: %q is also listed in kinds", i, name)
		}
	}
	return nil
}

func (k *Kind) validate(path string) error {
	if !kindNameRE.MatchString(k.Kind) {
		return fmt.Errorf("%s.kind: invalid kind name %q", path, k.Kind)
	}
	if k.Group == "" {
		return fmt.Errorf("%s.group: required", path)
	}
	if k.Version == "" {
		return fmt.Errorf("%s.version: required", path)
	}
	if len(k.APIPath) == 0 || k.APIPath[0] != '/' {
		return fmt.Errorf("%s.apiPath: must start with '/', got %q", path, k.APIPath)
	}
	if len(k.Scopes) == 0 {
		return fmt.Errorf("%s.scopes: at least one scope required", path)
	}
	seenScope := make(map[string]struct{}, len(k.Scopes))
	for i, s := range k.Scopes {
		sp := fmt.Sprintf("%s.scopes[%d]", path, i)
		if _, ok := knownScopes[s]; !ok {
			return fmt.Errorf("%s: unknown scope %q", sp, s)
		}
		if _, dup := seenScope[s]; dup {
			return fmt.Errorf("%s: duplicate scope %q", sp, s)
		}
		seenScope[s] = struct{}{}
	}
	if k.WaitForReady && !k.HasStatus {
		return fmt.Errorf("%s.waitForReady: requires hasStatus", path)
	}
	if !k.Typed {
		if k.TerraformName != "" {
			return fmt.Errorf("%s.terraformName: must be empty for manifest-only kinds", path)
		}
		if k.Spec != nil {
			return k.Spec.validate(path + ".spec")
		}
		return nil
	}
	if !terraformNameRE.MatchString(k.TerraformName) {
		return fmt.Errorf("%s.terraformName: invalid name %q", path, k.TerraformName)
	}
	if k.Spec == nil {
		return fmt.Errorf("%s.spec: required for typed kinds", path)
	}
	if k.Spec.Type != "object" {
		return fmt.Errorf("%s.spec.type: must be \"object\", got %q", path, k.Spec.Type)
	}
	return k.Spec.validate(path + ".spec")
}

func (f *Field) validate(path string) error {
	if _, ok := knownTypes[f.Type]; !ok {
		return fmt.Errorf("%s.type: unknown type %q", path, f.Type)
	}
	if len(f.Properties) > 0 || len(f.PropertyOrder) > 0 {
		if f.Type != "object" {
			return fmt.Errorf("%s.properties: only allowed when type is \"object\"", path)
		}
	}
	if f.Items != nil && f.Type != "array" {
		return fmt.Errorf("%s.items: only allowed when type is \"array\"", path)
	}
	if f.Values != nil && f.Type != "map" {
		return fmt.Errorf("%s.values: only allowed when type is \"map\"", path)
	}
	if f.Type == "array" && f.Items == nil {
		return fmt.Errorf("%s.items: required when type is \"array\"", path)
	}
	if f.Type == "map" && f.Values == nil {
		return fmt.Errorf("%s.values: required when type is \"map\"", path)
	}
	if err := f.validateProperties(path); err != nil {
		return err
	}
	if f.Items != nil {
		if err := f.Items.validate(path + ".items"); err != nil {
			return err
		}
	}
	if f.Values != nil {
		if err := f.Values.validate(path + ".values"); err != nil {
			return err
		}
	}
	return f.validateConstraints(path)
}

func (f *Field) validateProperties(path string) error {
	if len(f.PropertyOrder) > 0 {
		if len(f.PropertyOrder) != len(f.Properties) {
			return fmt.Errorf("%s.properties: propertyOrder has %d names for %d properties", path, len(f.PropertyOrder), len(f.Properties))
		}
		for _, name := range f.PropertyOrder {
			if _, ok := f.Properties[name]; !ok {
				return fmt.Errorf("%s.properties: propertyOrder names %q which is not a property", path, name)
			}
		}
	}
	names := make([]string, 0, len(f.Properties))
	for name := range f.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		childPath := path + ".properties." + name
		if !propertyNameRE.MatchString(name) {
			return fmt.Errorf("%s: invalid property name", childPath)
		}
		child := f.Properties[name]
		if child == nil {
			return fmt.Errorf("%s: nil field", childPath)
		}
		if err := child.validate(childPath); err != nil {
			return err
		}
	}
	return nil
}

func (f *Field) validateConstraints(path string) error {
	if f.MinLength != nil && *f.MinLength < 0 {
		return fmt.Errorf("%s.minLength: must be >= 0", path)
	}
	if f.MaxLength != nil && *f.MaxLength < 0 {
		return fmt.Errorf("%s.maxLength: must be >= 0", path)
	}
	if f.MinLength != nil && f.MaxLength != nil && *f.MinLength > *f.MaxLength {
		return fmt.Errorf("%s.minLength: greater than maxLength", path)
	}
	if f.Minimum != nil && f.Maximum != nil && *f.Minimum > *f.Maximum {
		return fmt.Errorf("%s.minimum: greater than maximum", path)
	}
	if f.Pattern != "" {
		if _, err := regexp.Compile(f.Pattern); err != nil {
			return fmt.Errorf("%s.pattern: invalid regular expression: %v", path, err)
		}
	}
	inEnum := make(map[string]struct{}, len(f.Enum))
	for _, v := range f.Enum {
		if _, dup := inEnum[v]; dup {
			return fmt.Errorf("%s.enum: duplicate value %q", path, v)
		}
		inEnum[v] = struct{}{}
	}
	for i, doc := range f.EnumDocs {
		if _, ok := inEnum[doc.Value]; !ok {
			return fmt.Errorf("%s.enumDocs[%d].value: %q is not in enum", path, i, doc.Value)
		}
	}
	return nil
}
