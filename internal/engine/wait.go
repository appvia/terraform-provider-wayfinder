// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Poll configures readiness polling: Initial doubles after each GET up to Maximum.
type Poll struct{ Initial, Maximum time.Duration }

// DefaultPoll is the production polling schedule (D-WAIT: 2s doubling to 15s).
func DefaultPoll() Poll { return Poll{Initial: 2 * time.Second, Maximum: 15 * time.Second} }

// Waiter polls an object until it is reconciled (D-WAIT) or gone (D-CRUD delete).
// API errors are reported against Target.
type Waiter struct {
	Client *client.Client
	Poll   Poll
	Target ErrorTarget
}

// Ready polls until the object is reconciled exactly as corev3.IsReconciled defines it, then
// maps the status: Success/Complete succeed, Warning/ActionRequired/ApprovalRequired warn,
// Failure/Error fail. It returns the last object read (nil if none).
func (w Waiter) Ready(ctx context.Context, ref client.Ref) (map[string]any, diag.Diagnostics) {
	return w.poll(ctx, ref, false)
}

// Gone polls until GET returns 404, or the context ends.
func (w Waiter) Gone(ctx context.Context, ref client.Ref) diag.Diagnostics {
	_, d := w.poll(ctx, ref, true)
	return d
}

func (r *kindResource) wait(ctx context.Context, ref client.Ref, deleting bool) (map[string]any, diag.Diagnostics) {
	w := Waiter{Client: r.client, Poll: Poll{Initial: r.poll.initial, Maximum: r.poll.maximum}, Target: r.errorTarget(ref.Name)}
	return w.poll(ctx, ref, deleting)
}

func (w Waiter) poll(ctx context.Context, ref client.Ref, deleting bool) (map[string]any, diag.Diagnostics) {
	var d diag.Diagnostics

	last := "Pending"

	interval := w.Poll.Initial
	for {
		obj, err := w.Client.Get(ctx, ref)
		if deleting && errors.Is(err, client.ErrNotFound) {
			return nil, d
		}

		if err != nil {
			if ctx.Err() != nil {
				d.AddError("Wayfinder operation timed out", fmt.Sprintf("Last status: %s. %s", last, context.Cause(ctx)))
			} else {
				AddAPIError(&d, err, w.Target)
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
					d.AddWarning(w.Target.Kind+" "+last, s.Status.Message)
					return obj, d
				case "Failure", "Error":
					d.AddError(w.Target.Kind+" "+last, s.Status.Message+"\n"+s.Status.Detail)
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

		interval = min(interval*2, w.Poll.Maximum)
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
