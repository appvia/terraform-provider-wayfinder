// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type secretInputs struct {
	config, plan, state tftypes.Value
	creating            bool
	path                string
	include             map[string]bool
}

func writeValues(in secretInputs) (tftypes.Value, error) {
	if in.plan.IsNull() || !in.plan.IsKnown() {
		return in.plan, nil
	}
	switch typ := in.plan.Type().(type) {
	case tftypes.Object, tftypes.Map:
		planned, err := attributes(in.plan)
		if err != nil {
			return tftypes.Value{}, err
		}
		configured, err := attributes(in.config)
		if err != nil {
			return tftypes.Value{}, err
		}
		stored, err := attributes(in.state)
		if err != nil {
			return tftypes.Value{}, err
		}
		for name, v := range planned {
			p := name
			if in.path != "" {
				switch typ.(type) {
				case tftypes.Map:
					p = fmt.Sprintf("%s[%q]", in.path, name)
				default:
					p = in.path + "." + name
				}
			}
			if strings.HasSuffix(name, "_wo") {
				if in.creating || !planned[name+"_version"].Equal(stored[name+"_version"]) {
					in.include[p] = true
					if c, ok := configured[name]; ok {
						planned[name] = c
					}
				}
				continue
			}
			nested, err := writeValues(secretInputs{config: configured[name], plan: v, state: stored[name], creating: in.creating, path: p, include: in.include})
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("%s: %w", p, err)
			}
			planned[name] = nested
		}
		return tftypes.NewValue(in.plan.Type(), planned), nil
	case tftypes.List:
		var planned, configured, stored []tftypes.Value
		if err := in.plan.As(&planned); err != nil {
			return tftypes.Value{}, fmt.Errorf("decode planned list: %w", err)
		}
		if in.config.Type() != nil && !in.config.IsNull() && in.config.IsKnown() {
			if err := in.config.As(&configured); err != nil {
				return tftypes.Value{}, fmt.Errorf("decode configured list: %w", err)
			}
		}
		if in.state.Type() != nil && !in.state.IsNull() && in.state.IsKnown() {
			if err := in.state.As(&stored); err != nil {
				return tftypes.Value{}, fmt.Errorf("decode stored list: %w", err)
			}
		}
		for i, v := range planned {
			var c, s tftypes.Value
			if i < len(configured) {
				c = configured[i]
			}
			if i < len(stored) {
				s = stored[i]
			}
			nested, err := writeValues(secretInputs{config: c, plan: v, state: s, creating: in.creating, path: fmt.Sprintf("%s[%d]", in.path, i), include: in.include})
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("list element %d: %w", i, err)
			}
			planned[i] = nested
		}
		return tftypes.NewValue(typ, planned), nil
	default:
		return in.plan, nil
	}
}

func attributes(v tftypes.Value) (map[string]tftypes.Value, error) {
	out := map[string]tftypes.Value{}
	if v.Type() == nil || v.IsNull() || !v.IsKnown() {
		return out, nil
	}
	if err := v.As(&out); err != nil {
		return nil, fmt.Errorf("decode attributes: %w", err)
	}
	return out, nil
}
