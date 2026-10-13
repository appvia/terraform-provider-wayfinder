package manifestresource

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"sigs.k8s.io/yaml"
)

var (
	errManifest  = errors.New("invalid manifest")
	documentSep  = regexp.MustCompile(`(?m)^---[ \t]*$`)
	identityKeys = []string{"name", "workspace", "environment", "version"}
)

// desired is one parsed manifest object bound to its kind. object has nulls removed and the
// canonical apiVersion <kind.Group>/<kind.Version>.
type desired struct {
	kind     manifest.Kind
	identity convert.Identity
	object   map[string]any
}

// importID is <group>/<Kind>/[workspace/[environment/]]name[@version].
func (d desired) importID() string { return formatID(d.kind, d.identity) }

func formatID(k manifest.Kind, i convert.Identity) string {
	return k.Group + "/" + k.Kind + "/" + i.ID()
}

// parseID resolves an import ID into its kind and identity.
func parseID(m *manifest.Manifest, id string) (manifest.Kind, convert.Identity, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return manifest.Kind{}, convert.Identity{}, fmt.Errorf("import ID %q must have the form <group>/<Kind>/[workspace/[environment/]]name[@version], for example org/Workspace/plat: %w", id, convert.ErrImportID)
	}
	k, err := resolveKind(m, parts[0], "", parts[1])
	if err != nil {
		return manifest.Kind{}, convert.Identity{}, err
	}
	i, err := convert.ParseImportID(k, parts[2])
	if err != nil {
		return manifest.Kind{}, convert.Identity{}, fmt.Errorf("import ID %q: %w", id, err)
	}
	return k, i, nil
}

// parseManifest decodes YAML or JSON holding exactly one Wayfinder object.
func parseManifest(m *manifest.Manifest, text string) (desired, error) {
	docs := 0
	for _, doc := range documentSep.Split(text, -1) {
		if strings.TrimSpace(doc) != "" {
			docs++
		}
	}
	if docs > 1 {
		return desired{}, fmt.Errorf("manifest holds %d YAML documents; use one wayfinder_manifest per object: %w", docs, errManifest)
	}
	raw, err := yaml.YAMLToJSON([]byte(text))
	if err != nil {
		return desired{}, fmt.Errorf("manifest is not valid YAML or JSON (%w): %w", err, errManifest)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return desired{}, fmt.Errorf("manifest must be one object with apiVersion, kind and metadata: %w", errManifest)
	}
	obj, _ = withoutNulls(obj).(map[string]any)
	apiVersion, _ := obj["apiVersion"].(string)
	group, version, err := splitAPIVersion(apiVersion)
	if err != nil {
		return desired{}, err
	}
	kindName, _ := obj["kind"].(string)
	if kindName == "" {
		return desired{}, fmt.Errorf("manifest kind must be a non-empty string: %w", errManifest)
	}
	k, err := resolveKind(m, group, version, kindName)
	if err != nil {
		return desired{}, err
	}
	i, err := identityOf(k, obj)
	if err != nil {
		return desired{}, err
	}
	obj["apiVersion"] = k.Group + "/" + k.Version
	return desired{kind: k, identity: i, object: obj}, nil
}

func identityOf(k manifest.Kind, obj map[string]any) (convert.Identity, error) {
	md, ok := obj["metadata"].(map[string]any)
	if !ok {
		return convert.Identity{}, fmt.Errorf("manifest needs a metadata object with a name: %w", errManifest)
	}
	values := map[string]string{}
	for _, key := range identityKeys {
		v, set := md[key]
		if !set {
			continue
		}
		s, isString := v.(string)
		if !isString || strings.ContainsAny(s, "/@") {
			return convert.Identity{}, fmt.Errorf("metadata.%s must be a string without '/' or '@': %w", key, errManifest)
		}
		values[key] = s
	}
	i := convert.Identity{Name: values["name"], Workspace: values["workspace"], Environment: values["environment"], Version: values["version"]}
	switch {
	case i.Name == "":
		return convert.Identity{}, fmt.Errorf("metadata.name is required: %w", errManifest)
	case i.Environment != "" && i.Workspace == "":
		return convert.Identity{}, fmt.Errorf("metadata.environment needs metadata.workspace: %w", errManifest)
	}
	if _, err := convert.ParseImportID(k, i.ID()); err != nil {
		return convert.Identity{}, fmt.Errorf("metadata does not match kind %s: %w", k.Kind, err)
	}
	return i, nil
}

// withoutNulls drops null map entries recursively; the API omits empty fields, so an explicit
// null would otherwise never match the server object.
func withoutNulls(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			if child != nil {
				out[k] = withoutNulls(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			out[i] = withoutNulls(child)
		}
		return out
	default:
		return v
	}
}
