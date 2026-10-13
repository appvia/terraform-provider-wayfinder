package engine

import (
	"encoding/json"
	"fmt"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type objectStatus struct {
	Metadata struct {
		Generation int64 `json:"generation"`
	} `json:"metadata"`
	Status struct {
		Status     string `json:"status"`
		Message    string `json:"message"`
		Detail     string `json:"detail"`
		Processing struct {
			OperationID string `json:"operationID"`
		} `json:"processing"`
		LastReconcile struct {
			Generation  int64  `json:"generation"`
			OperationID string `json:"operationID"`
		} `json:"lastReconcile"`
	} `json:"status"`
}

func inspectObject(obj map[string]any) (objectStatus, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return objectStatus{}, fmt.Errorf("encode object metadata: %w", err)
	}
	var s objectStatus
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("decode object metadata: %w", err)
	}
	return s, nil
}

func identityOf(k manifest.Kind, v tftypes.Value) (convert.Identity, error) {
	var attrs map[string]tftypes.Value
	if err := v.As(&attrs); err != nil {
		return convert.Identity{}, fmt.Errorf("decode identity: %w", err)
	}
	values := map[string]string{}
	for _, name := range []string{"name", "workspace", "environment", "version"} {
		a, ok := attrs[name]
		if !ok || a.IsNull() || !a.IsKnown() {
			continue
		}
		var value string
		if err := a.As(&value); err != nil {
			return convert.Identity{}, fmt.Errorf("decode %s: %w", name, err)
		}
		values[name] = value
	}
	i := convert.Identity{Name: values["name"], Workspace: values["workspace"], Environment: values["environment"], Version: values["version"]}
	return convert.ParseImportID(k, i.ID())
}

func reference(k manifest.Kind, i convert.Identity) client.Ref {
	return client.Ref{Group: k.Group, Version: k.Version, APIPath: k.APIPath, Workspace: i.Workspace, Environment: i.Environment, Name: i.Name, ObjectVersion: i.Version, Versioned: k.Versioned}
}

// Reference builds the client reference for an identity of kind k.
func Reference(k manifest.Kind, i convert.Identity) client.Ref { return reference(k, i) }

// Generation returns metadata.generation of a server object.
func Generation(obj map[string]any) (int64, error) {
	s, err := inspectObject(obj)
	return s.Metadata.Generation, err
}
