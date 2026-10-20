// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert_test

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func fixtureValue(typ tftypes.Type, text string) (tftypes.Value, error) {
	var raw any
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return tftypes.Value{}, fmt.Errorf("fixture JSON: %w", err)
	}

	return fixtureNode(typ, raw)
}

func fixtureNode(typ tftypes.Type, raw any) (tftypes.Value, error) {
	if raw == nil {
		return tftypes.NewValue(typ, nil), nil
	}

	value := raw

	switch t := typ.(type) {
	case tftypes.Object:
		attrs := map[string]tftypes.Value{}

		m := raw.(map[string]any)
		for name, child := range t.AttributeTypes {
			v, err := fixtureNode(child, m[name])
			if err != nil {
				return tftypes.Value{}, err
			}

			attrs[name] = v
		}

		value = attrs
	case tftypes.Map:
		attrs := map[string]tftypes.Value{}

		for name, raw := range raw.(map[string]any) {
			v, err := fixtureNode(t.ElementType, raw)
			if err != nil {
				return tftypes.Value{}, err
			}

			attrs[name] = v
		}

		value = attrs
	case tftypes.List:
		elems := []tftypes.Value{}

		for _, raw := range raw.([]any) {
			v, err := fixtureNode(t.ElementType, raw)
			if err != nil {
				return tftypes.Value{}, err
			}

			elems = append(elems, v)
		}

		value = elems
	}

	if err := tftypes.ValidateValue(typ, value); err != nil {
		return tftypes.Value{}, fmt.Errorf("fixture value: %w", err)
	}

	return tftypes.NewValue(typ, value), nil
}
