// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"testing"
)

func TestAPIInfo_returns_endpoints_when_unauthenticated(t *testing.T) {
	// Given
	s := newServer(t)
	// When
	got := callWithToken(t, s, "", http.MethodGet, "/apiinfo", nil)
	// Then
	wantCode(t, got, http.StatusOK)

	if got.body["generalAPI"] != "/api/v3/general" || got.body["resourceAPI"] != "/api/v3/resources" || got.body["ui"] != "https://portal.example/" {
		t.Fatalf("apiinfo = %s", got.raw)
	}
}
