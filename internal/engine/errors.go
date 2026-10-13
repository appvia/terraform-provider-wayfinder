package engine

import (
	"errors"
	"fmt"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func (r *kindResource) apiDiagnostics(d *diag.Diagnostics, err error, id string) {
	addr := "wayfinder_" + typeSuffix(r.kind) + ".<name>"
	var validation *client.ValidationError
	var api *client.APIError
	switch {
	case errors.Is(err, client.ErrAlreadyExists):
		d.AddError("Resource already exists", fmt.Sprintf("A %s named %s already exists. Import it with: terraform import %s %s", r.kind.Kind, id, addr, id))
	case errors.Is(err, client.ErrModified):
		d.AddError("Resource changed outside Terraform", fmt.Sprintf("%s %s was changed outside Terraform since Terraform last wrote it. To keep the outside change, re-import it (terraform state rm %s, then terraform import %s %s) and apply again.", r.kind.Kind, id, addr, addr, id))
	case errors.As(err, &validation):
		if len(validation.Fields) == 0 {
			d.AddError("Wayfinder validation failed", validation.Message)
			return
		}
		for _, f := range validation.Fields {
			p, ok := r.mapping.AttributePath(f.Field)
			if ok {
				d.AddAttributeError(p, "Wayfinder validation failed", f.Message)
			} else {
				d.AddError("Wayfinder validation failed", f.Message)
			}
		}
	case errors.As(err, &api):
		d.AddError("Wayfinder request failed", api.Message)
	default:
		d.AddError("Wayfinder request failed", err.Error())
	}
}
