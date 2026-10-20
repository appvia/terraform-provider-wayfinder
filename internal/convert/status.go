// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert

import "encoding/json"

// StatusSummary extracts the CommonStatus strings, ignoring extended status.
func StatusSummary(obj map[string]any) (status, message, detail string) {
	s, _ := obj["status"].(map[string]any)
	status, _ = s["status"].(string)
	message, _ = s["message"].(string)
	detail, _ = s["detail"].(string)

	return
}

// StatusJSON returns normalized full status JSON, or false if absent/invalid.
func StatusJSON(obj map[string]any) (string, bool) {
	status, ok := obj["status"]
	if !ok || status == nil {
		return "", false
	}

	b, err := json.Marshal(status)
	if err != nil {
		return "", false
	}

	return string(b), true
}
