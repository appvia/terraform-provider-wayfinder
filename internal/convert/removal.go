// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func removedValue(plan tftypes.Value, present bool, prior tftypes.Value) bool {
	return prior.Type() != nil && prior.IsKnown() && !prior.IsNull() && (!present || plan.IsKnown() && plan.IsNull())
}

func removeMapKeys(base map[string]any, plan, prior tftypes.Value) error {
	if plan.Type() != nil && !plan.IsKnown() {
		return nil
	}

	old, err := valueMap(prior)
	if err != nil {
		return fmt.Errorf("prior map: %w", err)
	}

	want, err := valueMap(plan)
	if err != nil {
		return fmt.Errorf("planned map: %w", err)
	}

	for key, value := range old {
		planned, present := want[key]
		if removedValue(planned, present, value) {
			delete(base, key)
		}
	}

	return nil
}

// List elements are replaced from the plan, except computed fields and blank
// secret placeholders needed by the server's keep-stored-secret contract.
func (n *node) retainedBase(base any) any {
	if n.computed || n.sensitive {
		return base
	}

	switch n.kind {
	case "object":
		out := map[string]any{}

		for _, child := range n.children {
			if value, exists := jsonMap(base)[child.jsonName]; exists {
				if retained := child.retainedBase(value); retained != nil {
					out[child.jsonName] = retained
				}
			}
		}

		return out
	case "map":
		out := map[string]any{}

		for key, value := range jsonMap(base) {
			if retained := n.element.retainedBase(value); retained != nil {
				out[key] = retained
			}
		}

		return out
	case "array":
		old, _ := base.([]any)

		out := make([]any, len(old))
		for i, value := range old {
			out[i] = n.element.retainedBase(value)
		}

		return out
	default:
		return nil
	}
}
