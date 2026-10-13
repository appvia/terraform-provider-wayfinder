package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func (r *kindResource) wait(ctx context.Context, ref client.Ref, deleting bool) (map[string]any, diag.Diagnostics) {
	var d diag.Diagnostics
	last := "Pending"
	interval := r.poll.initial
	for {
		obj, err := r.client.Get(ctx, ref)
		if deleting && errors.Is(err, client.ErrNotFound) {
			return nil, d
		}
		if err != nil {
			if ctx.Err() != nil {
				d.AddError("Wayfinder operation timed out", fmt.Sprintf("Last status: %s. %s", last, context.Cause(ctx)))
			} else {
				r.apiDiagnostics(&d, err, ref.Name)
			}
			return obj, d
		}
		if !deleting {
			s, err := inspectObject(obj)
			if err != nil {
				d.AddError("Invalid Wayfinder status", err.Error())
				return obj, d
			}
			last = s.Status.Status
			if s.Metadata.Generation == s.Status.LastReconcile.Generation && s.Status.Processing.OperationID == s.Status.LastReconcile.OperationID {
				switch last {
				case "Success", "Complete":
					return obj, d
				case "Warning", "ActionRequired", "ApprovalRequired":
					d.AddWarning(r.kind.Kind+" "+last, s.Status.Message)
					return obj, d
				case "Failure", "Error":
					d.AddError(r.kind.Kind+" "+last, s.Status.Message+"\n"+s.Status.Detail)
					return obj, d
				}
			}
		} else {
			last = "deleting"
		}
		if err := pollDelay(ctx, interval); err != nil {
			d.AddError("Wayfinder operation timed out", fmt.Sprintf("Last status: %s. %s", last, err))
			return obj, d
		}
		interval = min(interval*2, r.poll.maximum)
	}
}

func pollDelay(ctx context.Context, delay time.Duration) error {
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting for Wayfinder: %w", context.Cause(ctx))
	case <-t.C:
		return nil
	}
}
