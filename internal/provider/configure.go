package provider

import (
	"context"
	"net"
	"net/http"
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
	httpClient := newHTTPClient()
	cfg.HTTP = httpClient
	resolved, err := auth.Resolve(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Unable to configure the Wayfinder provider", err.Error())
		return
	}
	c := &client.Client{
		BaseURL:     resolved.Server,
		Tenant:      resolved.Tenant,
		TokenSource: resolved.TokenSource,
		HTTP:        httpClient,
		UserAgent:   client.UserAgent(p.version),
	}
	resp.ResourceData = c
	resp.DataSourceData = c
}

// authConfig maps the provider block onto auth.Config. Null attributes become "" so auth falls
// back to the environment; unknown attributes cannot be resolved and are reported.
func authConfig(m providerModel) (auth.Config, diag.Diagnostics) {
	var d diag.Diagnostics
	value := func(name string, v types.String) string {
		if v.IsUnknown() {
			d.AddAttributeError(path.Root(name), "Unknown Wayfinder provider attribute",
				"The provider cannot be configured because "+name+" is not known until apply. "+
					"Set it to a value known at plan time, or use the matching WAYFINDER_* environment variable.")
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
	}
	return cfg, d
}

func newHTTPClient() *http.Client {
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
		},
	}
}
