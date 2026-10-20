// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"encoding/json"
	"strings"
)

// deepCopy returns an independent copy of a JSON object, normalising Go values (ints, typed
// maps) to their decoded-JSON form so stored objects always look like request bodies.
func deepCopy(obj map[string]any) map[string]any {
	b, err := json.Marshal(obj)
	if err != nil {
		panic("fakeapi: object is not JSON-serialisable: " + err.Error())
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		panic("fakeapi: " + err.Error())
	}
	return out
}

// child returns m[key] as an object, creating it when create is set.
func child(m map[string]any, key string, create bool) map[string]any {
	c, ok := m[key].(map[string]any)
	if !ok && create {
		c = map[string]any{}
		m[key] = c
	}
	return c
}

func metadata(obj map[string]any) map[string]any { return child(obj, "metadata", true) }

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func generation(obj map[string]any) int64 {
	switch v := metadata(obj)["generation"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

// eachLeaf calls fn(parent, key) for every location the dotted path resolves to. Array values
// apply the remaining path to each element; missing intermediate objects stop the walk.
func eachLeaf(v any, parts []string, fn func(parent map[string]any, key string)) {
	switch node := v.(type) {
	case []any:
		for _, el := range node {
			eachLeaf(el, parts, fn)
		}
	case map[string]any:
		if len(parts) == 1 {
			fn(node, parts[0])
			return
		}
		eachLeaf(node[parts[0]], parts[1:], fn)
	}
}

// eachLeafPair walks dst and src in parallel along the path; arrays are aligned by index.
// src may be nil where the stored object has nothing at that location.
func eachLeafPair(dst, src any, parts []string, fn func(d, s map[string]any, key string)) {
	switch node := dst.(type) {
	case []any:
		srcArr, _ := src.([]any)
		for i, el := range node {
			var se any
			if i < len(srcArr) {
				se = srcArr[i]
			}
			eachLeafPair(el, se, parts, fn)
		}
	case map[string]any:
		srcMap, _ := src.(map[string]any)
		if len(parts) == 1 {
			fn(node, srcMap, parts[0])
			return
		}
		var next any
		if srcMap != nil {
			next = srcMap[parts[0]]
		}
		eachLeafPair(node[parts[0]], next, parts[1:], fn)
	}
}

func split(path string) []string { return strings.Split(path, ".") }

// sanitize blanks every +encrypted leaf: strings become "" and map values become "" with keys
// kept (pkg/wfsecrets/encrypted_fields.go:80-86, applied on read at
// pkg/stores/wfclientstore/wfclientstore.go:313-318).
// ASSUMPTION: write responses (POST/PUT/DELETE) are blanked too, since every generic handler
// sanitises before writing (pkg/apihandlers/types.go:160-177) and secrets must never echo back.
func sanitize(k *KindSpec, obj map[string]any) {
	for _, p := range k.SensitiveStrings {
		eachLeaf(obj, split(p), func(m map[string]any, key string) {
			if _, ok := m[key].(string); ok {
				m[key] = ""
			}
		})
	}
	for _, p := range k.SensitiveMaps {
		eachLeaf(obj, split(p), func(m map[string]any, key string) {
			if vals, ok := m[key].(map[string]any); ok {
				for vk := range vals {
					vals[vk] = ""
				}
			}
		})
	}
}

// mergeSecrets backfills a PUT body from the stored object exactly as mergeEncryptedLeaf does
// (pkg/wfsecrets/encrypted_fields.go:389-428; caller pkg/apihandlers/annotation_validation.go:360-380):
//   - string: blank (or absent) takes the stored value; there is no way to clear it;
//   - map: the SUBMITTED key set is authoritative; a blank value keeps the stored one, a
//     non-blank value rotates it, and a key left out is dropped (stored keys never return).
//
// ASSUMPTION: the walk stops where the submitted body has no parent object (Wayfinder's nil
// pointer case); arrays are matched by index.
func mergeSecrets(k *KindSpec, dst, src map[string]any) {
	for _, p := range k.SensitiveStrings {
		eachLeafPair(dst, src, split(p), func(d, s map[string]any, key string) {
			cur, _ := d[key].(string)
			if stored, ok := s[key].(string); ok && cur == "" {
				d[key] = stored
			}
		})
	}
	for _, p := range k.SensitiveMaps {
		eachLeafPair(dst, src, split(p), func(d, s map[string]any, key string) {
			vals, _ := d[key].(map[string]any)
			stored, _ := s[key].(map[string]any)
			for vk, v := range vals {
				if v != "" {
					continue
				}
				if sv, ok := stored[vk]; ok {
					vals[vk] = sv
				}
			}
		})
	}
}

// applyDefaults fills server-defaulted fields that the client left unset (creating parents).
func applyDefaults(k *KindSpec, obj map[string]any) {
	for path, val := range k.Defaults {
		parts := split(path)
		m := obj
		for _, p := range parts[:len(parts)-1] {
			m = child(m, p, true)
		}
		last := parts[len(parts)-1]
		if _, set := m[last]; !set {
			m[last] = deepCopy(map[string]any{"v": val})["v"]
		}
	}
}
