// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// These wire types mirror the credential subset of wf's profiles file.
type cliConfig struct {
	Users          map[string]cliUser           `json:"users"`
	CurrentProfile string                       `json:"current-profile"`
	Profiles       map[string]*cliStoredProfile `json:"profiles"`
	Servers        map[string]cliServer         `json:"servers"`
}

type cliUser struct {
	Identity *cliIdentity `json:"identity"`
	Token    *string      `json:"token"`
}

type cliIdentity struct {
	RefreshToken string              `json:"refresh-token"`
	Token        string              `json:"token"`
	ClientID     string              `json:"client-id"`
	ClientSecret string              `json:"client-secret"`
	Federated    *cliStoredFederated `json:"federated-identity"`
}

type cliStoredProfile struct {
	User   string `json:"user"`
	Server string `json:"server"`
	Tenant string `json:"tenant"`
}

type cliServer struct {
	Server        string `json:"server"`
	CACertificate string `json:"caCertificate"`
}

type cliStoredFederated struct {
	// wf serializes ServiceAccountRef as an object, not its String() form.
	ServiceAccount struct {
		Tenant    string `json:"tenant"`
		Workspace string `json:"workspace"`
		Name      string `json:"name"`
	} `json:"service-account"`
	SubjectTokenFile string `json:"subject-token-file"`
	TokenServer      string `json:"token-server"`
	TokenServerToken string `json:"token-server-token"`
	Audience         string `json:"audience"`
}

type cliProfile struct {
	Name, Server, Tenant                        string
	CAPEM                                       []byte
	RefreshToken, Token, ClientID, ClientSecret string
	Federated                                   *cliFederated
}

type cliFederated struct {
	ServiceAccount, SubjectTokenFile, TokenServer, TokenServerToken, Audience string
}

func cliConfigPath(getenv func(string) string) (string, error) {
	path := os.Expand(getenv("WAYFINDER_CONFIG"), getenv)
	if path == "" {
		path = os.Expand(filepath.Join(cliHomeDir(runtime.GOOS, getenv), ".wayfinder", "profiles"), getenv)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", &safeError{operation: "resolve wf CLI profiles file path", cause: err}
	}
	return absolute, nil
}

// cliHomeDir follows wf's OS-specific home precedence with an injected environment.
func cliHomeDir(goos string, getenv func(string) string) string {
	if goos == "windows" {
		home := getenv("HOMEDRIVE") + getenv("HOMEPATH")
		if home == "" {
			home = getenv("USERPROFILE")
		}
		return home
	}
	return getenv("HOME")
}

// loadCLIProfile only reads the file; it never updates credentials or CLI defaults.
func loadCLIProfile(path, name string) (cliProfile, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cliProfile{}, &cliProfileNotFoundError{path: path, cause: err}
	}
	if err != nil {
		return cliProfile{}, &safeError{operation: "read wf CLI profiles file " + path, cause: err}
	}
	var cfg cliConfig
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return cliProfile{}, &safeError{operation: "parse wf CLI profiles file " + path, cause: err}
	}
	if name == "" {
		name = cfg.CurrentProfile
	}
	if name == "" {
		return cliProfile{}, &configurationError{message: `no wf CLI profile selected - set cli_profile / WAYFINDER_CLI_PROFILE, or run "wf use profile NAME"`}
	}
	profile := cfg.Profiles[name]
	if profile == nil {
		names := make([]string, 0, len(cfg.Profiles))
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		return cliProfile{}, &configurationError{message: fmt.Sprintf("wf CLI profile %q not found in %s (available: %s)", name, path, strings.Join(names, ", "))}
	}
	server := cfg.Servers[profile.Server]
	if server.Server == "" {
		return cliProfile{}, &configurationError{message: fmt.Sprintf(`wf CLI profile %q has no server - run "wf login --profile %s"`, name, name)}
	}
	resolved := cliProfile{Name: name, Server: server.Server, Tenant: profile.Tenant, CAPEM: []byte(server.CACertificate)}
	user := cfg.Users[profile.User]
	if identity := user.Identity; identity != nil {
		resolved.RefreshToken, resolved.Token = identity.RefreshToken, identity.Token
		resolved.ClientID, resolved.ClientSecret = identity.ClientID, identity.ClientSecret
		if federated := identity.Federated; federated != nil {
			ref := federated.ServiceAccount
			parts := make([]string, 0, 3)
			if ref.Tenant != "" {
				parts = append(parts, ref.Tenant)
			}
			if ref.Workspace != "" {
				parts = append(parts, ref.Workspace)
			}
			parts = append(parts, ref.Name)
			// An empty federated-identity block carries no service account, so it is not a credential.
			if ref.Tenant != "" || ref.Workspace != "" || ref.Name != "" {
				resolved.Federated = &cliFederated{
					ServiceAccount: strings.Join(parts, ":"), SubjectTokenFile: federated.SubjectTokenFile,
					TokenServer: federated.TokenServer, TokenServerToken: federated.TokenServerToken, Audience: federated.Audience,
				}
			}
		}
	}
	if resolved.Token == "" && user.Token != nil {
		resolved.Token = *user.Token
	}
	if resolved.RefreshToken == "" && (resolved.ClientID == "" || resolved.ClientSecret == "") && resolved.Federated == nil && resolved.Token == "" {
		return cliProfile{}, &configurationError{message: fmt.Sprintf(`wf CLI profile %q has no stored credentials - run "wf login --profile %s"`, name, name)}
	}
	return resolved, nil
}

type cliProfileNotFoundError struct {
	path  string
	cause error
}

func (e *cliProfileNotFoundError) Error() string {
	return fmt.Sprintf(`wf CLI profiles file %s not found - sign in with "wf login", or set WAYFINDER_CONFIG`, e.path)
}

func (e *cliProfileNotFoundError) Unwrap() error { return e.cause }
