package manifestresource

import (
	"context"
	"errors"
	"strconv"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const generationKey = "generation"

// privateState is the subset of framework private state used for the generation baseline.
type privateState interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func setGeneration(ctx context.Context, p privateState, obj map[string]any, d *diag.Diagnostics) {
	g, err := engine.Generation(obj)
	if err != nil {
		d.AddError("Invalid Wayfinder metadata", err.Error())
		return
	}
	d.Append(p.SetKey(ctx, generationKey, []byte(strconv.FormatInt(g, 10)))...)
}

func errorTarget(k manifest.Kind, i convert.Identity) engine.ErrorTarget {
	return engine.ErrorTarget{Kind: k.Kind, ID: formatID(k, i), Address: "wayfinder_manifest.<name>"}
}

// apiError reports field validation errors on manifest_wo when the server field is a value in
// wo, otherwise on manifest, naming the server field; every other error is reported exactly as
// the engine does.
func apiError(d *diag.Diagnostics, err error, t engine.ErrorTarget, wo map[string]any) {
	var validation *client.ValidationError
	if errors.As(err, &validation) && len(validation.Fields) > 0 {
		for _, f := range validation.Fields {
			attr := path.Root("manifest")
			if p, perr := parseWOPath(f.Field); perr == nil {
				if _, found := lookup(wo, p); found {
					attr = path.Root("manifest_wo")
				}
			}
			d.AddAttributeError(attr, "Wayfinder validation failed", f.Field+": "+f.Message)
		}
		return
	}
	engine.AddAPIError(d, err, t)
}

func statusValue(obj map[string]any) types.Object {
	s, m, detail := convert.StatusSummary(obj)
	return types.ObjectValueMust(statusTypes, map[string]attr.Value{"status": types.StringValue(s), "message": types.StringValue(m), "detail": types.StringValue(detail)})
}

func (r *manifestResource) waiter(k manifest.Kind, i convert.Identity) engine.Waiter {
	return engine.Waiter{Client: r.client, Poll: r.poll, Target: errorTarget(k, i)}
}

// shouldWait applies D-WAIT: only kinds with a reconciling controller, unless disabled.
func shouldWait(k manifest.Kind, m model) bool {
	return k.HasStatus && k.WaitForReady && !m.WaitForReady.Equal(types.BoolValue(false))
}
