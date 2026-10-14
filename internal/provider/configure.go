package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appvia/terraform-provider-wayfinder/internal/auth"
	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

// requestTimeout bounds one API or token request; readiness waits are bounded separately by
// resource timeouts.
const requestTimeout = 60 * time.Second

func (p *wayfinderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var m providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, d := authConfig(m)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	httpClient, err := NewHTTPClient(os.Getenv(CAFileEnv))
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure the Wayfinder provider", err.Error())
		return
	}
	cfg.HTTP = httpClient
	cfg.CLIHTTP = CLIHTTPClient
	cfg.Context = ctx
	resolved, err := auth.Resolve(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure the Wayfinder provider", err.Error())
		return
	}
	c := &client.Client{
		BaseURL:     resolved.Server,
		Tenant:      resolved.Tenant,
		TokenSource: resolved.TokenSource,
		HTTP:        resolved.HTTP,
		UserAgent:   client.UserAgent(p.version),
	}
	resp.ResourceData = c
	resp.DataSourceData = c
}

// authConfig maps the provider block onto auth.Config. Null attributes become "" so auth falls
// back to the environment; unknown attributes cannot be resolved and are reported.
func authConfig(m providerModel) (auth.Config, diag.Diagnostics) {
	var d diag.Diagnostics
	unknown := func(name string) {
		d.AddAttributeError(path.Root(name), "Unknown Wayfinder provider attribute",
			"The provider cannot be configured because "+name+" is not known until apply. "+
				"Set it to a value known at plan time, or use the matching WAYFINDER_* environment variable.")
	}
	value := func(name string, v types.String) string {
		if v.IsUnknown() {
			unknown(name)
		}
		return v.ValueString()
	}
	cfg := auth.Config{
		Server:                        value("server", m.Server),
		Tenant:                        value("tenant", m.Tenant),
		Token:                         value("token", m.Token),
		ClientID:                      value("client_id", m.ClientID),
		ClientSecret:                  value("client_secret", m.ClientSecret),
		ServiceAccount:                value("service_account", m.ServiceAccount),
		FederatedTokenFile:            value("federated_token_file", m.FederatedTokenFile),
		FederatedTokenRequestURL:      value("federated_token_request_url", m.FederatedTokenRequestURL),
		FederatedTokenRequestToken:    value("federated_token_request_token", m.FederatedTokenRequestToken),
		FederatedTokenRequestAudience: value("federated_token_request_audience", m.FederatedTokenRequestAudience),
		CLIProfile:                    value("cli_profile", m.CLIProfile),
	}
	switch {
	case m.UseCLI.IsUnknown():
		unknown("use_cli")
	case !m.UseCLI.IsNull():
		useCLI := m.UseCLI.ValueBool()
		cfg.UseCLI = &useCLI
	}
	return cfg, d
}

// CAFileEnv names a PEM bundle of extra certificate authorities to trust, for private and test
// deployments whose API certificate is issued by a custom certificate authority. It is
// read from the environment only: Go ignores SSL_CERT_FILE on macOS and Windows.
const CAFileEnv = "WAYFINDER_CA_FILE"

// NewHTTPClient builds the API and token client; caFile, when set, adds its certificates to
// the system roots.
func NewHTTPClient(caFile string) (*http.Client, error) {
	return newHTTPClient(caFile, nil)
}

// CLIHTTPClient builds the API and token client for a wf CLI profile, trusting the profile's CA
// certificate in addition to the system roots and WAYFINDER_CA_FILE.
func CLIHTTPClient(caPEM []byte) (*http.Client, error) {
	return newHTTPClient(os.Getenv(CAFileEnv), caPEM)
}

func newHTTPClient(caFile string, extraPEM []byte) (*http.Client, error) {
	var tlsConfig *tls.Config
	if caFile != "" || len(extraPEM) > 0 {
		pool, err := caPool(caFile, extraPEM)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: requestTimeout,
			ExpectContinueTimeout: time.Second,
			TLSClientConfig:       tlsConfig,
		},
	}, nil
}

func caPool(caFile string, extraPEM []byte) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", CAFileEnv, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("%s %s contains no PEM certificates", CAFileEnv, caFile)
		}
	}
	if len(extraPEM) > 0 && !pool.AppendCertsFromPEM(extraPEM) {
		return nil, errors.New("the wf CLI profile's CA certificate contains no PEM certificates")
	}
	return pool, nil
}
