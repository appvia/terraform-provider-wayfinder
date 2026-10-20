// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const cliProfilesFixture = `
current-profile: staging
users:
  alice:
    identity:
      refresh-token: fake-refresh-secret
      token: fake-cached-token
  robot:
    identity:
      client-id: acme:robot:key
      client-secret: fake-client-secret
  ci:
    identity:
      federated-identity:
        service-account:
          tenant: acme
          workspace: build
          environment: ""
          name: ci
        subject-token-file: /fake/subject
        token-server: https://oidc.example
        token-server-token: fake-oidc-secret
        audience: https://portal.example
  bearer:
    identity: {token: fake-identity-token}
    token: fake-legacy-token
  legacy:
    token: fake-static-token
profiles:
  staging: {user: alice, server: staging-api, tenant: acme, workspace: ignored, environment: ignored}
  production: {user: robot, server: production-api, tenant: production}
  federation: {user: ci, server: staging-api, tenant: acme}
  bearer: {user: bearer, server: staging-api, tenant: acme}
  legacy: {user: legacy, server: staging-api, tenant: acme}
servers:
  staging-api:
    server: https://staging.example/
    caCertificate: |
      -----BEGIN CERTIFICATE-----
      fake-ca
      -----END CERTIFICATE-----
  production-api: {server: https://production.example}
`

func writeCLIProfiles(t *testing.T, contents string) string {
	t.Helper()

	var dir string

	t.Cleanup(func() {
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("temporary profiles directory was not removed: %v", err)
		}
	})
	dir = t.TempDir()

	path := filepath.Join(dir, "profiles")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestCLIProfile_whenStoredCredentialsSelected(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		want cliProfile
	}{
		{"", cliProfile{Name: "staging", Server: "https://staging.example/", Tenant: "acme", RefreshToken: "fake-refresh-secret", Token: "fake-cached-token"}},
		{"staging", cliProfile{Name: "staging", Server: "https://staging.example/", Tenant: "acme", RefreshToken: "fake-refresh-secret", Token: "fake-cached-token"}},
		{"production", cliProfile{Name: "production", Server: "https://production.example", Tenant: "production", ClientID: "acme:robot:key", ClientSecret: "fake-client-secret"}},
		{"federation", cliProfile{Name: "federation", Server: "https://staging.example/", Tenant: "acme", Federated: &cliFederated{
			ServiceAccount: "acme:build:ci", SubjectTokenFile: "/fake/subject", TokenServer: "https://oidc.example", TokenServerToken: "fake-oidc-secret", Audience: "https://portal.example",
		}}},
		{"bearer", cliProfile{Name: "bearer", Server: "https://staging.example/", Tenant: "acme", Token: "fake-identity-token"}},
		{"legacy", cliProfile{Name: "legacy", Server: "https://staging.example/", Tenant: "acme", Token: "fake-static-token"}},
	} {
		name := tc.name
		if name == "" {
			name = "current-profile"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Given
			path := writeCLIProfiles(t, cliProfilesFixture)

			want := tc.want
			if want.Server == "https://staging.example/" {
				want.CAPEM = []byte("-----BEGIN CERTIFICATE-----\nfake-ca\n-----END CERTIFICATE-----\n")
			}
			// When
			got, err := loadCLIProfile(path, tc.name)
			// Then
			if err != nil {
				t.Fatal(err)
			}

			if got.Name != want.Name || got.Server != want.Server || got.Tenant != want.Tenant ||
				got.RefreshToken != want.RefreshToken || got.Token != want.Token ||
				got.ClientID != want.ClientID || got.ClientSecret != want.ClientSecret ||
				!bytes.Equal(got.CAPEM, want.CAPEM) || !reflect.DeepEqual(got.Federated, want.Federated) {
				t.Fatal("resolved profile did not match selected credentials, endpoint, tenant and CA")
			}

			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, []byte(cliProfilesFixture)) {
				t.Fatalf("profiles file changed: %v", err)
			}
		})
	}
}

func TestCLIProfile_whenConfigurationInvalid(t *testing.T) {
	t.Parallel()

	const secret = "credential-must-never-appear"
	for _, tc := range []struct {
		name, contents, selection, want string
		parseError                      bool
	}{
		{"malformed YAML", "users: [" + secret, "", "parse wf CLI profiles file %s failed", true},
		{"malformed credential type", "users: {alice: {identity: {refresh-token: [" + secret + "]}}}", "", "parse wf CLI profiles file %s failed", true},
		{"malformed federated reference", "users: {alice: {identity: {federated-identity: {service-account: [" + secret + "]}}}}", "", "parse wf CLI profiles file %s failed", true},
		{"no selection", "users: {alice: {token: " + secret + "}}", "", `no wf CLI profile selected - set cli_profile / WAYFINDER_CLI_PROFILE, or run "wf use profile NAME"`, false},
		{"unknown profile sorted", cliProfilesFixture, "absent", `wf CLI profile "absent" not found in %s (available: bearer, federation, legacy, production, staging)`, false},
		{"missing server entry", "current-profile: test\nprofiles: {test: {server: absent}}", "", `wf CLI profile "test" has no server - run "wf login --profile test"`, false},
		{"empty server endpoint", "current-profile: test\nprofiles: {test: {server: empty}}\nservers: {empty: {}}", "", `wf CLI profile "test" has no server - run "wf login --profile test"`, false},
		{"null server", "current-profile: test\nprofiles: {test: {server: empty}}\nservers: {empty: null}", "", `wf CLI profile "test" has no server - run "wf login --profile test"`, false},
		{"missing user", "current-profile: test\nprofiles: {test: {server: api, user: absent}}\nservers: {api: {server: https://api.example}}", "", `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`, false},
		{"null user", "current-profile: test\nprofiles: {test: {server: api, user: alice}}\nservers: {api: {server: https://api.example}}\nusers: {alice: null}", "", `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`, false},
		{"empty static token", "current-profile: test\nprofiles: {test: {server: api, user: alice}}\nservers: {api: {server: https://api.example}}\nusers: {alice: {token: ''}}", "", `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`, false},
		{"empty federated identity", "current-profile: test\nprofiles: {test: {server: api, user: alice}}\nservers: {api: {server: https://api.example}}\nusers: {alice: {identity: {federated-identity: {}}}}", "", `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`, false},
		{"partial client credentials", "current-profile: test\nprofiles: {test: {server: api, user: alice}}\nservers: {api: {server: https://api.example}}\nusers: {alice: {identity: {client-secret: " + secret + "}}}", "", `wf CLI profile "test" has no stored credentials - run "wf login --profile test"`, false},
		{"null selected profile", "current-profile: test\nprofiles: {test: null}", "", `wf CLI profile "test" not found in %s (available: test)`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			path := writeCLIProfiles(t, tc.contents)

			want := tc.want
			if strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, path)
			}
			// When
			_, err := loadCLIProfile(path, tc.selection)
			// Then
			if err == nil {
				t.Fatal("invalid profiles were accepted")
			}

			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "fake-") {
				t.Fatal("profile error leaked credentials")
			}

			if err.Error() != want {
				t.Fatalf("error = %q, want %q", err.Error(), want)
			}

			if tc.parseError {
				var safe *safeError
				if !errors.As(err, &safe) || errors.Unwrap(safe) == nil {
					t.Fatal("parse error lost its safe wrapper or cause")
				}
			}
		})
	}
}

func TestCLIProfile_whenFileMissing(t *testing.T) {
	t.Parallel()
	// Given
	path := filepath.Join(t.TempDir(), "missing")
	// When
	_, err := loadCLIProfile(path, "")
	// Then
	want := fmt.Sprintf(`wf CLI profiles file %s not found - sign in with "wf login", or set WAYFINDER_CONFIG`, path)
	if err == nil || !errors.Is(err, fs.ErrNotExist) || err.Error() != want {
		t.Fatalf("missing file error = %v", err)
	}
}

func TestCLIProfile_whenFileUnreadable(t *testing.T) {
	t.Parallel()
	// Given a directory cannot be read as a profiles file, even when tests run as root.
	path := t.TempDir()
	// When
	_, err := loadCLIProfile(path, "")
	// Then
	var safe *safeError
	if !errors.As(err, &safe) || errors.Unwrap(safe) == nil {
		t.Fatalf("read error lost its safe wrapper or cause: %v", err)
	}
}

func TestCLIProfilePath_whenEnvironmentInjected(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"override expands HOME", map[string]string{"WAYFINDER_CONFIG": "$HOME/custom/profiles", "HOME": "/fake/home"}, filepath.FromSlash("/fake/home/custom/profiles")},
		{"override expands named variable", map[string]string{"WAYFINDER_CONFIG": "${PROFILE_DIR}/profiles", "PROFILE_DIR": "/fake/config"}, filepath.FromSlash("/fake/config/profiles")},
		{"relative override absolute", map[string]string{"WAYFINDER_CONFIG": "relative/profiles"}, filepath.Join("relative", "profiles")},
		{"expanded empty override uses default", map[string]string{"WAYFINDER_CONFIG": "$UNSET", "HOME": "/fake/home", "USERPROFILE": "/fake/home"}, filepath.Join("/fake/home", ".wayfinder", "profiles")},
		{"default home path", map[string]string{"HOME": "/fake/home", "USERPROFILE": "/fake/home"}, filepath.Join("/fake/home", ".wayfinder", "profiles")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			want, err := filepath.Abs(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			// When
			got, err := cliConfigPath(func(key string) string { return tc.values[key] })
			// Then
			if err != nil || got != want {
				t.Fatalf("path = %q, err = %v; want %q", got, err, want)
			}
		})
	}
}

func TestCLIProfileHome_whenOperatingSystemSelectsEnvironment(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, goos string
		values     map[string]string
		want       string
	}{
		{"Unix ignores USERPROFILE", "darwin", map[string]string{"HOME": "/unix", "USERPROFILE": `C:\Users\fallback`}, "/unix"},
		{"Windows drive and path beat USERPROFILE and HOME", "windows", map[string]string{"HOMEDRIVE": "D:", "HOMEPATH": `\Users\primary`, "USERPROFILE": `C:\Users\fallback`, "HOME": "/unix"}, `D:\Users\primary`},
		{"Windows USERPROFILE beats HOME", "windows", map[string]string{"USERPROFILE": `C:\Users\fallback`, "HOME": "/unix"}, `C:\Users\fallback`},
		{"Unix empty HOME does not use USERPROFILE", "linux", map[string]string{"USERPROFILE": `C:\Users\fallback`}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given
			getenv := func(key string) string { return tc.values[key] }
			// When
			got := cliHomeDir(tc.goos, getenv)
			// Then
			if got != tc.want {
				t.Fatalf("home = %q, want %q", got, tc.want)
			}

			if tc.goos == runtime.GOOS {
				path, err := cliConfigPath(getenv)

				want, absErr := filepath.Abs(filepath.Join(tc.want, ".wayfinder", "profiles"))
				if err != nil || absErr != nil || path != want {
					t.Fatalf("default path = %q, err = %v; want %q, err = %v", path, err, want, absErr)
				}
			}
		})
	}
}
