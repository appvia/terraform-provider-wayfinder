package engine

import (
	"context"
	"errors"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *kindResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.prepared(&resp.Diagnostics) {
		return
	}
	i, err := identityOf(r.kind, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource identity", err.Error())
		return
	}
	duration, d := r.operationTimeout(ctx, req.State.Raw, "delete")
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	ref := reference(r.kind, i)
	err = r.client.Delete(ctx, ref)
	if errors.Is(err, client.ErrNotFound) {
		return
	}
	if err != nil {
		r.apiDiagnostics(&resp.Diagnostics, err, i.ID())
		return
	}
	_, d = r.wait(ctx, ref, true)
	resp.Diagnostics.Append(d...)
}
