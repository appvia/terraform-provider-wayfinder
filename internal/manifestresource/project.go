package manifestresource

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
)

// project keeps only the parts of server that the configured shape names: map keys present in
// shape (recursively) and, for lists, each element projected onto the element at the same
// index. Extra list elements and values of a different JSON type are kept whole so that they
// still show as drift. Server defaults, status and server-added labels therefore never diff.
func project(server, shape any) any {
	switch s := shape.(type) {
	case map[string]any:
		sm, ok := server.(map[string]any)
		if !ok {
			return server
		}
		out := make(map[string]any, len(s))
		for k, child := range s {
			if v, present := sm[k]; present {
				out[k] = project(v, child)
			}
		}
		return out
	case []any:
		sl, ok := server.([]any)
		if !ok {
			return server
		}
		out := make([]any, len(sl))
		for i, v := range sl {
			out[i] = v
			if i < len(s) {
				out[i] = project(v, s[i])
			}
		}
		return out
	default:
		return server
	}
}

// overlay returns fresh with desired applied: maps merge recursively, other values are
// replaced, and keys that were in the previously applied manifest (prior) but are no longer
// configured are removed. Server-owned data (generation, status, server-added labels) and
// blank secrets, which the server keeps, are left as read.
func overlay(fresh, want, prior map[string]any) map[string]any {
	out := maps.Clone(fresh)
	if out == nil {
		out = map[string]any{}
	}
	for k := range prior {
		if _, kept := want[k]; !kept {
			delete(out, k)
		}
	}
	for k, v := range want {
		wm, wantMap := v.(map[string]any)
		fm, freshMap := out[k].(map[string]any)
		if wantMap && freshMap {
			pm, _ := prior[k].(map[string]any)
			out[k] = overlay(fm, wm, pm)
			continue
		}
		out[k] = v
	}
	return out
}

// importedView is the server object without status and server-owned metadata; it stands in
// for the manifest until configuration is applied after an import.
func importedView(obj map[string]any) map[string]any {
	out := maps.Clone(obj)
	delete(out, "status")
	if md, ok := out["metadata"].(map[string]any); ok {
		md = maps.Clone(md)
		for _, k := range []string{"generation", "uid", "creationTimestamp", "deletionTimestamp", "finalizers", "owner"} {
			delete(md, k)
		}
		for _, field := range []string{"labels", "annotations"} {
			if m, ok := md[field].(map[string]any); ok {
				m = maps.Clone(m)
				maps.DeleteFunc(m, func(k string, _ any) bool { return convert.IsServerManagedMetadataKey(k) })
				if len(m) == 0 {
					delete(md, field)
					continue
				}
				md[field] = m
			}
		}
		out["metadata"] = md
	}
	return out
}

func encodeJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode object: %w", err)
	}
	return string(b), nil
}

func decodeJSON(s string) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode object_json: %w", err)
	}
	return out, nil
}

// projectObject projects a server object onto want. A server apiVersion naming the same group
// and version (short or fully-qualified group) is reported in want's canonical form.
func projectObject(server, want map[string]any) map[string]any {
	out, _ := project(normalized(server), want).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	got, _ := server["apiVersion"].(string)
	canonical, _ := want["apiVersion"].(string)
	g, v, err := splitAPIVersion(got)
	cg, cv, cerr := splitAPIVersion(canonical)
	if err == nil && cerr == nil && v == cv && groupMatches(cg, g) {
		out["apiVersion"] = canonical
	}
	return out
}

// matches reports whether server, projected onto want, equals want.
func matches(server, want map[string]any) bool {
	return reflect.DeepEqual(projectObject(server, want), want)
}

// normalized round-trips through JSON so numbers and nested types compare like parsed YAML.
func normalized(obj map[string]any) map[string]any {
	b, err := json.Marshal(obj)
	if err != nil {
		return obj
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		return obj
	}
	return out
}
