package fakeapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

func TestVars_per_key_api_at_every_scope_masks_secrets(t *testing.T) {
	scopes := map[string]fakeapi.VarScope{
		"/api/v3/general/tenants/acme/vars":                                      {Tenant: "acme"},
		"/api/v3/general/tenants/acme/workspaces/ws1/vars":                       {Tenant: "acme", Workspace: "ws1"},
		"/api/v3/general/tenants/acme/workspaces/ws1/envs/dev/vars":              {Tenant: "acme", Workspace: "ws1", Environment: "dev"},
		"/api/v3/general/tenants/acme/workspaces/ws1/envs/dev/instancevars/app1": {Tenant: "acme", Workspace: "ws1", Environment: "dev", Instance: "app1"},
	}
	for base, scope := range scopes {
		t.Run(base, func(t *testing.T) {
			s := newServer(t)
			wantCode(t, call(t, s, http.MethodDelete, base+"/variables/none", nil), http.StatusNotFound)

			setVar := call(t, s, http.MethodPut, base+"/variables/region", map[string]any{"value": "eu-west-2"})
			setSec := call(t, s, http.MethodPut, base+"/secrets/db", map[string]any{"value": map[string]any{"password": "pw"}})
			getVar := call(t, s, http.MethodGet, base+"/variables/region", nil)
			getSec := call(t, s, http.MethodGet, base+"/secrets/db", nil)
			list := call(t, s, http.MethodGet, base, nil)

			wantCode(t, setVar, http.StatusOK)
			if setVar.body["key"] != "region" || setSec.body["key"] != "db" {
				t.Errorf("set responses = %s / %s", setVar.raw, setSec.raw)
			}
			if getVar.body["value"] != "eu-west-2" {
				t.Errorf("variable = %s", getVar.raw)
			}
			if getSec.body["key"] != "db" || getSec.body["value"] != nil {
				t.Errorf("secret not masked: %s", getSec.raw)
			}
			secs, _ := list.body["secrets"].([]any)
			vars, _ := list.body["variables"].([]any)
			if len(secs) != 1 || dig(secs[0].(map[string]any), "value") != nil || len(vars) != 1 {
				t.Errorf("list = %s", list.raw)
			}
			raw, ok := s.Secret(scope, "db")
			if !ok || !jsonEqual(raw, `{"password":"pw"}`) {
				t.Errorf("recorded secret = %s", raw)
			}

			wantCode(t, call(t, s, http.MethodDelete, base+"/secrets/db", nil), http.StatusNoContent)
			wantCode(t, call(t, s, http.MethodGet, base+"/secrets/db", nil), http.StatusNotFound)
			if _, ok := s.Secret(scope, "db"); ok {
				t.Error("secret still recorded after delete")
			}
		})
	}
}

func jsonEqual(raw json.RawMessage, want string) bool {
	var a, b any
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal([]byte(want), &b) != nil {
		return false
	}
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
