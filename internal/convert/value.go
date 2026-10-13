package convert

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func resourceAttrs(m *Mapping, v tftypes.Value) (map[string]tftypes.Value, error) {
	if v.Type() == nil {
		return nil, fmt.Errorf("missing resource type: %w", ErrValue)
	}
	if !v.Type().Equal(m.typ) {
		return nil, fmt.Errorf("resource type does not match mapping: %w", ErrValue)
	}
	return valueMap(v)
}

func valueMap(v tftypes.Value) (map[string]tftypes.Value, error) {
	out := map[string]tftypes.Value{}
	if v.Type() == nil || v.IsNull() || !v.IsKnown() {
		return out, nil
	}
	if err := v.As(&out); err != nil {
		return nil, fmt.Errorf("decode attributes: %w", err)
	}
	return out, nil
}

func jsonMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func cloneJSON(v any) (any, error) {
	switch x := v.(type) {
	case nil, string, bool, float64, float32, int, int64, int32, uint, uint64, uint32, json.Number:
		return x, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, key := range slices.Sorted(maps.Keys(x)) {
			value, err := cloneJSON(x[key])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			out[key] = value
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, value := range x {
			cloned, err := cloneJSON(value)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			out[i] = cloned
		}
		return out, nil
	default:
		// Base is JSON, but callers may use typed maps, RawMessage or numbers.
		// Round-tripping these values detaches their backing storage as well.
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("encode JSON: %w", err)
		}
		var out any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("decode JSON: %w", err)
		}
		return out, nil
	}
}

func newValue(typ tftypes.Type, v any) (tftypes.Value, error) {
	if err := tftypes.ValidateValue(typ, v); err != nil {
		return tftypes.Value{}, fmt.Errorf("validate Terraform value: %w", err)
	}
	return tftypes.NewValue(typ, v), nil
}

func readScalar(typ tftypes.Type, obj any) (tftypes.Value, error) {
	if obj == nil {
		return tftypes.NewValue(typ, nil), nil
	}
	if typ.Is(tftypes.Number) {
		switch obj.(type) {
		case json.Number, float64, float32, int, int64, int32, uint, uint64, uint32:
		default:
			return tftypes.Value{}, fmt.Errorf("expected JSON number: %w", ErrValue)
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("encode number: %w", err)
		}
		f, _, err := big.ParseFloat(string(b), 10, 512, big.ToNearestEven)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("parse number: %w", err)
		}
		return newValue(typ, f)
	}
	if t, ok := typ.(tftypes.Map); ok {
		m, ok := obj.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected JSON map: %w", ErrValue)
		}
		attrs := map[string]tftypes.Value{}
		for _, key := range slices.Sorted(maps.Keys(m)) {
			v, err := readScalar(t.ElementType, m[key])
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("[%q]: %w", key, err)
			}
			attrs[key] = v
		}
		return newValue(typ, attrs)
	}
	return newValue(typ, obj)
}
