// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (r *kindResource) operationTimeout(ctx context.Context, value tftypes.Value, operation string) (time.Duration, diag.Diagnostics) {
	var v timeouts.Value
	s := tfsdk.Config{Raw: value, Schema: r.schema}
	d := s.GetAttribute(ctx, path.Root("timeouts"), &v)
	if d.HasError() {
		return 0, d
	}
	switch operation {
	case "create":
		return v.Create(ctx, 30*time.Minute)
	case "update":
		return v.Update(ctx, 30*time.Minute)
	default:
		return v.Delete(ctx, 30*time.Minute)
	}
}

func (r *kindResource) shouldWait(v tftypes.Value) bool {
	if !r.kind.HasStatus || !r.kind.WaitForReady {
		return false
	}
	a, err := attributes(v)
	if err != nil {
		return true
	}
	w := a["wait_for_ready"]
	if w.IsNull() || !w.IsKnown() {
		return true
	}
	var enabled bool
	if err := w.As(&enabled); err != nil {
		return true
	}
	return enabled
}
