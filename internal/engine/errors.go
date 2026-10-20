// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"errors"
	"fmt"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// ErrorTarget names the object an API error refers to.
type ErrorTarget struct {
	// Kind is the Wayfinder kind name; ID is the Terraform import ID.
	Kind, ID string
	// Address is the resource address shown in import instructions, e.g. wayfinder_workspace.<name>.
	Address string
	// AttributePath maps a server validation field (e.g. spec.endpoint) to an attribute.
	// Nil, or a false result, reports the field message without an attribute path.
	AttributePath func(field string) (path.Path, bool)
}

// AddAPIError converts a client error into diagnostics: already exists and changed outside
// Terraform get import instructions, validation errors one diagnostic per field, and any
// other server message is passed through verbatim.
func AddAPIError(d *diag.Diagnostics, err error, t ErrorTarget) {
	var (
		validation *client.ValidationError
		api        *client.APIError
	)
	switch {
	case errors.Is(err, client.ErrAlreadyExists):
		d.AddError("Resource already exists", fmt.Sprintf("A %s named %s already exists. Import it with: terraform import %s %s", t.Kind, t.ID, t.Address, t.ID))
	case errors.Is(err, client.ErrModified):
		d.AddError("Resource changed outside Terraform", fmt.Sprintf("%s %s was changed outside Terraform since Terraform last wrote it. To keep the outside change, re-import it (terraform state rm %s, then terraform import %s %s) and apply again.", t.Kind, t.ID, t.Address, t.Address, t.ID))
	case errors.As(err, &validation):
		if len(validation.Fields) == 0 {
			d.AddError("Wayfinder validation failed", validation.Message)
			return
		}
		for _, f := range validation.Fields {
			if t.AttributePath != nil {
				if p, ok := t.AttributePath(f.Field); ok {
					d.AddAttributeError(p, "Wayfinder validation failed", f.Message)
					continue
				}
			}
			d.AddError("Wayfinder validation failed", f.Message)
		}
	case errors.As(err, &api):
		d.AddError("Wayfinder request failed", api.Message)
	default:
		d.AddError("Wayfinder request failed", err.Error())
	}
}

func (r *kindResource) errorTarget(id string) ErrorTarget {
	return ErrorTarget{Kind: r.kind.Kind, ID: id, Address: "wayfinder_" + typeSuffix(r.kind) + ".<name>", AttributePath: r.mapping.AttributePath}
}

func (r *kindResource) apiDiagnostics(d *diag.Diagnostics, err error, id string) {
	AddAPIError(d, err, r.errorTarget(id))
}
