package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// providerModel is the provider block. Every attribute is optional and falls back to the
// identically named WAYFINDER_* environment variable (plan R11).
type providerModel struct {
	Server                        types.String `tfsdk:"server"`
	Tenant                        types.String `tfsdk:"tenant"`
	Token                         types.String `tfsdk:"token"`
	ClientID                      types.String `tfsdk:"client_id"`
	ClientSecret                  types.String `tfsdk:"client_secret"`
	ServiceAccount                types.String `tfsdk:"service_account"`
	FederatedTokenFile            types.String `tfsdk:"federated_token_file"`
	FederatedTokenRequestURL      types.String `tfsdk:"federated_token_request_url"`
	FederatedTokenRequestToken    types.String `tfsdk:"federated_token_request_token"`
	FederatedTokenRequestAudience types.String `tfsdk:"federated_token_request_audience"`
}

// attribute documents one provider attribute.
type attribute struct {
	name, env, doc string
	sensitive      bool
}

var attributes = []attribute{
	{name: "server", env: "WAYFINDER_SERVER", doc: "Wayfinder API endpoint. Defaults to `https://api.on.wayfinder.run`."},
	{name: "tenant", env: "WAYFINDER_TENANT", doc: "Wayfinder tenant to manage. When unset, it is derived from `service_account` (`tenant:[workspace:]name`) or from the `client_id` prefix before the first `:`."},
	{name: "token", env: "WAYFINDER_TOKEN", doc: "Wayfinder access token. Takes precedence over `service_account` and client credentials.", sensitive: true},
	{name: "client_id", env: "WAYFINDER_CLIENT_ID", doc: "Service account client ID, used with `client_secret`."},
	{name: "client_secret", env: "WAYFINDER_CLIENT_SECRET", doc: "Service account client secret, used with `client_id`.", sensitive: true},
	{name: "service_account", env: "WAYFINDER_SERVICE_ACCOUNT", doc: "Fully qualified service account (`tenant:name` or `tenant:workspace:name`) to authenticate as with a federated identity token. Takes precedence over client credentials."},
	{name: "federated_token_file", env: "WAYFINDER_FEDERATED_TOKEN_FILE", doc: "Path to a file holding the federated identity token exchanged for `service_account`."},
	{name: "federated_token_request_url", env: "WAYFINDER_FEDERATED_TOKEN_REQUEST_URL", doc: "URL to request the federated identity token from. Inside GitHub Actions, `ACTIONS_ID_TOKEN_REQUEST_URL` is used when this is unset."},
	{name: "federated_token_request_token", env: "WAYFINDER_FEDERATED_TOKEN_REQUEST_TOKEN", doc: "Bearer token for `federated_token_request_url`. Inside GitHub Actions, `ACTIONS_ID_TOKEN_REQUEST_TOKEN` is used when the request URL is unset.", sensitive: true},
	{name: "federated_token_request_audience", env: "WAYFINDER_FEDERATED_TOKEN_REQUEST_AUDIENCE", doc: "Audience requested for the federated identity token."},
}

const providerDescription = "Manage Wayfinder resources.\n\n" +
	"Credentials are taken from the provider block or the matching `WAYFINDER_*` environment variables, " +
	"in this order of precedence: `token`, then `service_account` with a federated identity token, then `client_id` and `client_secret`. " +
	"The provider does not read `wf` CLI profiles (`~/.wayfinder`), and `WAYFINDER_WORKSPACE` and `WAYFINDER_ENVIRONMENT` " +
	"are not used as defaults: set `workspace` and `environment` on each resource."

func providerSchema() schema.Schema {
	attrs := make(map[string]schema.Attribute, len(attributes))
	for _, a := range attributes {
		attrs[a.name] = schema.StringAttribute{
			Optional:            true,
			Sensitive:           a.sensitive,
			MarkdownDescription: a.doc + " Can also be set with the `" + a.env + "` environment variable.",
		}
	}
	return schema.Schema{MarkdownDescription: providerDescription, Attributes: attrs}
}
