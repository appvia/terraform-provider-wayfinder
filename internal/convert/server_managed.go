// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import "slices"

// serverManagedMetadataKeys are label/annotation keys Wayfinder stamps itself on every
// write (observed on real servers: createdBy, lastUpdated, lastUpdatedBy, changed). They
// are not user intent, so import hydration must not adopt them. This is the single list;
// it is deliberately exact rather than an "appvia.io/" prefix, because other appvia.io/
// keys (e.g. resourceSuffix, legacyOIDCAudience) can be set by users or tools.
var serverManagedMetadataKeys = []string{
	"appvia.io/createdBy",
	"appvia.io/lastUpdated",
	"appvia.io/lastUpdatedBy",
	"appvia.io/changed",
}

// IsServerManagedMetadataKey reports whether key is a label/annotation key stamped by the server.
func IsServerManagedMetadataKey(key string) bool {
	return slices.Contains(serverManagedMetadataKeys, key)
}

// withoutServerManaged returns a copy of the metadata map m without server-managed keys.
// Non-map values are returned unchanged; a map left empty becomes nil (null, as when the
// server returned none).
func withoutServerManaged(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := make(map[string]any, len(m))
	for k, val := range m {
		if !IsServerManagedMetadataKey(k) {
			out[k] = val
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
