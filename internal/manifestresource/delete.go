package manifestresource

import (
	"context"
	"errors"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Delete sends DELETE (404 counts as deleted) and waits until the object is gone, since
// objects with finalizers stay readable while Wayfinder cleans up.
func (r *manifestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.ready(&resp.Diagnostics, true) {
		return
	}
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, i, err := parseID(r.manifest, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Invalid resource identity", err.Error())
		return
	}
	duration, d := state.Timeouts.Delete(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	ref := engine.Reference(k, i)
	err = r.client.Delete(ctx, ref)
	if errors.Is(err, client.ErrNotFound) {
		return
	}
	if err != nil {
		apiError(&resp.Diagnostics, err, errorTarget(k, i), nil)
		return
	}
	resp.Diagnostics.Append(r.waiter(k, i).Gone(ctx, ref)...)
}

// ImportState accepts <group>/<Kind>/[workspace/[environment/]]name[@version].
func (r *manifestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.ready(&resp.Diagnostics, false) {
		return
	}
	k, i, err := parseID(r.manifest, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), formatID(k, i))...)
}
