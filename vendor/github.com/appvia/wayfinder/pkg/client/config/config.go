/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"crypto/tls"
	"encoding/pem"
	"errors"
	"io"
	"net/url"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/config"
	"github.com/appvia/wayfinder/pkg/version"
	"github.com/appvia/wayfinder/server/org/services/authentication"
)

// New creates a configuration
func New(reader io.Reader) (*Config, error) {
	conf := &Config{}

	if err := yaml.NewDecoder(reader).Decode(conf); err != nil {
		return nil, err
	}

	if conf.FeatureGates == nil {
		conf.FeatureGates = map[string]bool{}
	}
	for fg, enabled := range config.DefaultFeatureGates() {
		if _, isSet := conf.FeatureGates[fg]; !isSet {
			conf.FeatureGates[fg] = enabled
		}
	}

	return conf, nil
}

// NewEmpty returns an empty configuration
func NewEmpty() *Config {
	return &Config{
		AuthInfos:    make(map[string]*AuthInfo),
		Profiles:     make(map[string]*Profile),
		Servers:      make(map[string]*Server),
		Version:      version.Release,
		FeatureGates: config.DefaultFeatureGates(),
	}
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	return nil
}

// IsAccessToken returns true if the current profile is for an access token user
func (c *Config) IsAccessToken() bool {
	ai := c.GetAuthInfo(c.CurrentProfile)
	if ai == nil {
		return false
	}
	return ai.Identity != nil && ai.Identity.IsExchangeToken()
}

// IsFeatureEnabled checks if the feature is enabled
func (c *Config) IsFeatureEnabled(feature string) bool {
	if c.FeatureGates == nil {
		return false
	}

	return c.FeatureGates[feature]
}

// ListFeatures returns a list of features flags
func (c *Config) ListFeatures() []string {
	var list []string

	for k := range config.DefaultFeatureGates() {
		list = append(list, k)
	}

	return list
}

// NewProfileWithAuth creates the profile
func (c *Config) NewProfileWithAuth(name, endpoint string, auth *AuthInfo) error {

	if c.HasProfile(name) {
		return errors.New("profile name already in use")
	}

	c.CreateProfile(name, endpoint)
	c.AddAuthInfo(name, auth)

	return nil
}

// CreateProfile is used to create a profile
func (c *Config) CreateProfile(name, endpoint string) {
	var ca []byte

	u, _ := url.Parse(endpoint)
	if u != nil && u.Scheme == "https" && u.Hostname() == "localhost" {
		ca = c.getUntrustedCA(strings.TrimPrefix(endpoint, "https://"))
	}

	c.AddProfile(name, &Profile{
		Server:   name,
		AuthInfo: name,
	})
	c.AddServer(name, &Server{Endpoint: endpoint, CACertificate: string(ca)})
}

func (c *Config) getUntrustedCA(url string) (ca []byte) {
	conn, err := tls.Dial("tcp", url, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		common.LogWithoutContext().Debugf("failed to connect to %s: %s", url, err.Error())
		return nil
	}

	if err := conn.Handshake(); err != nil {
		common.LogWithoutContext().Debugf("SSL handshake failed with %s: %s", url, err.Error())
		return nil
	}

	l := len(conn.ConnectionState().PeerCertificates)
	caCert := conn.ConnectionState().PeerCertificates[0]
	cert := conn.ConnectionState().PeerCertificates[l-1]
	for _, domain := range cert.DNSNames {
		if domain == "localhost" {
			block := &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: caCert.Raw,
			}
			return pem.EncodeToMemory(block)
		}
	}

	return nil
}

// ListProfiles returns a list of profile names
func (c *Config) ListProfiles() []string {
	if c.Profiles == nil {
		return nil
	}
	var list []string

	for k := range c.Profiles {
		list = append(list, k)
	}

	return list
}

// GetProfile returns the profile
func (c *Config) GetProfile(name string) *Profile {
	if !c.HasProfile(name) {
		return &Profile{}
	}

	return c.Profiles[name]
}

// GetProfileAuthMethod returns the method of authentication for a profile
func (c *Config) GetProfileAuthMethod(name string) string {
	if !c.HasProfile(name) {
		return ""
	}
	if !c.HasAuthInfo(c.Profiles[name].AuthInfo) {
		return ""
	}
	auth := c.AuthInfos[c.Profiles[name].AuthInfo]
	switch {
	case auth.Token != nil:
		return "token"
	case auth.Identity != nil:
		return "idtoken"
	case auth.OIDC != nil:
		return "oidc"
	}

	return "none"
}

// IsExchangeToken is used to check with the authentication is an exchange token
func (k *Identity) IsExchangeToken() bool {
	return IsExchangeToken([]byte(k.RefreshToken))
}

// IsExpired checks if the access token is expired
func (k *Identity) IsExpired() (bool, error) {
	return authentication.IsTokenExpired(k.Token)
}

// GetServer returns the endpoint for the profile
func (c *Config) GetServer(name string) *Server {
	if !c.HasProfile(name) {
		return &Server{}
	}

	return c.Servers[c.Profiles[name].Server]
}

// GetAuthInfo returns the auth for a profile
func (c *Config) GetAuthInfo(name string) *AuthInfo {
	ct := c.Profiles[name]
	if ct == nil {
		return &AuthInfo{}
	}

	a := c.AuthInfos[ct.AuthInfo]

	if a == nil {
		return &AuthInfo{}
	}

	return a
}

// HasAuth checks if we have auth enabled
func (c *Config) HasAuth(name string) bool {
	a := c.GetAuthInfo(name)
	if a.Token != nil || a.Identity != nil || a.OIDC != nil {
		return true
	}

	return false
}

// AddProfile adds a profile to the config
func (c *Config) AddProfile(name string, ctx *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[name] = ctx
}

// AddServer adds a server
func (c *Config) AddServer(name string, server *Server) {
	if c.Servers == nil {
		c.Servers = make(map[string]*Server)
	}
	server.Endpoint = strings.TrimSuffix(server.Endpoint, "/")
	c.Servers[name] = server
}

// AddAuthInfo adds a authentication
func (c *Config) AddAuthInfo(name string, auth *AuthInfo) {
	if c.AuthInfos == nil {
		c.AuthInfos = make(map[string]*AuthInfo)
	}
	c.AuthInfos[name] = auth
}

// HasValidProfile checks we have a current context
func (c *Config) HasValidProfile(name string) error {
	if name == "" {
		return ErrNoProfileSelected
	}
	if !c.HasServer(c.Profiles[name].Server) {
		return ErrNoProfileEndpoint
	}

	return nil
}

// HasProfile checks if the context exists in the config
func (c *Config) HasProfile(name string) bool {
	_, found := c.Profiles[name]

	return found
}

// HasServer checks if the context exists in the config
func (c *Config) HasServer(name string) bool {
	_, found := c.Servers[name]

	return found
}

// HasAuthInfo checks if the context exists in the config
func (c *Config) HasAuthInfo(name string) bool {
	_, found := c.AuthInfos[name]

	return found
}

// RemoveServer removes a server instance
func (c *Config) RemoveServer(name string) {
	delete(c.Servers, name)
}

// RemoveUserInfo removes the user info
func (c *Config) RemoveUserInfo(name string) {
	delete(c.AuthInfos, name)
}

// RemoveProfile removes the profile
func (c *Config) RemoveProfile(name string) {
	p, found := c.Profiles[name]
	if !found {
		return
	}
	c.RemoveServer(p.Server)
	c.RemoveUserInfo(p.AuthInfo)

	delete(c.Profiles, name)
}

// Update writes the config to the file
func (c *Config) Update(w io.Writer) error {
	return yaml.NewEncoder(w).Encode(c)
}
