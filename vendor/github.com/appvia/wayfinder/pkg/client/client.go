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

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/apiserver/types"
	"github.com/appvia/wayfinder/pkg/client/config"
	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils/httputils"
	jwsutils "github.com/appvia/wayfinder/pkg/utils/jwt"
	"github.com/appvia/wayfinder/pkg/utils/openid"
	"github.com/appvia/wayfinder/pkg/utils/retry"
	"github.com/appvia/wayfinder/pkg/utils/validation"
	"github.com/appvia/wayfinder/pkg/version"
)

// apiClient implements the api and raw client
type apiClient struct {
	// body is the response body
	body *bytes.Buffer
	// ctx is the context for the client
	ctx context.Context
	// cfg is the client configuration
	cfg *config.Config
	// authtoken is a authorization header
	authtoken string
	// ferror is used to handle errors in the method chain
	ferror error
	// endpoint is the raw endpoint template to use
	endpoint string
	// handler is an update handler for the config
	handler UpdateHandlerFunc
	// hc is the http client to use
	hc *http.Client
	// profile is the name of the profile to use
	profile string
	// parameters hold the parameters for the request
	parameters map[string]string
	// payload is the outbound payload
	payload interface{}
	// result is what we decode into
	result interface{}
	// queryparams are a collection of query parameters
	queryparams url.Values
	// follow indiates we convert to websocket and follow stream
	follow bool
	// client is a reference to the main client interface
	client Interface
	// unauthenticated indicates no need to add auth
	unauthenticated bool
}

var (
	// ErrAuthenticationRequired requires authentication
	ErrAuthenticationRequired = types.Error{
		Code:    http.StatusUnauthorized,
		Message: "authentication required",
	}
)

// cc provides a wrapper around th config
type cc struct {
	cfg       *config.Config
	handler   UpdateHandlerFunc
	profile   string
	apiClient func(cfg *config.Config) RestInterface
}

// New creates and returns an API client
func New(cfg *config.Config, options ...OptionFunc) (Interface, error) {
	if cfg == nil {
		return nil, errors.New("no client configuration")
	}

	c := &cc{cfg: cfg}

	// apply the options
	for _, fn := range options {
		fn(c)
	}
	if c.apiClient == nil {
		c.apiClient = func(cfg *config.Config) RestInterface {
			return &apiClient{
				cfg:        cfg,
				client:     c,
				handler:    c.handler,
				parameters: make(map[string]string),
				profile:    c.CurrentProfile(),
			}
		}
	}

	return c, nil
}

// Config return a copy of the client configuration
func (c *cc) Config() *config.Config {
	return c.cfg
}

// OverrideProfile sets the default profile to use
func (c *cc) OverrideProfile(name string) Interface {
	c.profile = name

	return c
}

// CurrentProfile returns the current profile
func (c *cc) CurrentProfile() string {
	if c.profile != "" {
		return c.profile
	}

	return c.cfg.CurrentProfile
}

// Request creates a request instance
func (c *cc) Request() RestInterface {
	return c.apiClient(c.cfg)
}

// RefreshIdentity is called to refresh the identity token of the client
func (c *cc) RefreshIdentity() error {

	auth := c.cfg.AuthInfos[c.CurrentProfile()]

	switch {
	case auth == nil:
		return NewProfileInvalidError("missing authentication profile", c.CurrentProfile())

	case auth.OIDC != nil:

		refresh, err := openid.DefaultTokenRefresher.RefreshToken(context.Background(),
			auth.OIDC.RefreshToken,
			auth.OIDC.TokenURL,
			auth.OIDC.ClientID,
			auth.OIDC.ClientSecret,
		)
		if err != nil {
			return err
		}

		auth.OIDC.AccessToken = refresh.AccessToken
		auth.OIDC.IDToken = refresh.IDToken

	case auth.Identity != nil:

		switch {
		case auth.Identity.IsExchangeToken():
			common.LogWithoutContext().Debug("attempting to refresh access token via access token exchange")

			ttl := 30 * time.Minute
			token, err := ExchangeAccessToken(c, []byte(auth.Identity.RefreshToken), ttl)
			if err != nil {
				common.LogWithoutContext().WithError(err).Error("trying to exchange access token")

				return err
			}
			auth.Identity.Token = string(token)

		default:
			common.LogWithoutContext().Debug("attempting to refresh identity token")

			token, err := RefreshWayfinderIdentityToken(c, []byte(auth.Identity.RefreshToken))
			if err != nil {
				return err
			}
			auth.Identity.Token = string(token)
		}

	default:
		return errors.New("no token available to refresh")
	}

	return c.handleConfigurationUpdate()
}

// handleConfigurationUpdate is called when the configuration has been updated
func (c *cc) handleConfigurationUpdate() error {
	if c.handler == nil {
		return nil
	}

	return c.handler()
}

func (a *apiClient) Profile() string {
	return a.profile
}

// makeAPIEndpoint is responsible for getting the api endpoint
func (a *apiClient) makeAPIEndpoint() (*config.Server, error) {
	// @step: check we have the endpoint
	profile, found := a.cfg.Profiles[a.Profile()]
	if !found {
		return nil, ErrMissingProfile
	}
	server, found := a.cfg.Servers[profile.Server]
	if !found {
		return nil, NewProfileInvalidError("missing profile server", a.Profile())
	}

	if server.Endpoint == "" {
		return nil, NewProfileInvalidError("missing endpoint", a.Profile())

	}

	return server, nil
}

// handleRequest is responsible for handling the request chain
func (a *apiClient) handleRequest(method string) RestInterface {
	err := func() error {
		if a.ferror != nil {
			return a.ferror
		}

		// @step: find the endpoint
		server, err := a.makeAPIEndpoint()
		if err != nil {
			a.ferror = err

			return a.ferror
		}

		if a.hc == nil {
			a.hc = a.MakeHTTPClient(server.CACertificate)
		}

		// @step: we generate the uri from the parameter
		uri, err := a.makeRequestURI()
		if err != nil {
			return err
		}
		common.LogWithoutContext().WithFields(map[string]interface{}{
			"cacert":   server.CACertificate != "",
			"endpoint": server.Endpoint,
			"method":   method,
			"uri":      uri,
		}).Debug("making request to wayfinder api")

		// @step: we generate the fully qualifies url
		ep := fmt.Sprintf("%s/%s", server.Endpoint, uri)

		now := time.Now()

		var resp *http.Response

		ctx := a.ctx
		if ctx == nil {
			ctx = context.Background()
		}

		err = retry.Retry(ctx, 3, true, 1*time.Second, func() (bool, error) {
			var respErr error
			resp, respErr = a.makeRequest(method, ep)
			if respErr != nil {
				return false, respErr
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				return false, nil
			}

			return true, nil
		})

		common.LogWithoutContext().WithField("time", time.Since(now).String()).Debug("processing time for request")

		if err != nil && err != retry.ErrReachMaxAttempts {
			return err
		}

		return a.handleResponse(resp)
	}()
	if err != nil {
		a.ferror = err
	}

	return a
}

// makeRequestURI is responsible for generating the request URI
func (a *apiClient) makeRequestURI() (string, error) {
	if a.endpoint == "" {
		return a.makeDefaultURL()
	}

	return a.makeEndpointURL()
}

// makeDefaultURL generates a URL from the /workspaces/<name>/resource/<name> format
func (a *apiClient) makeDefaultURL() (string, error) {
	var paths []string
	// @logic: we simply iterate in a known order of things i.e. workspace, resource
	// and name check if the parameter is there and if so append

	if value, found := a.HasParameter("workspace"); found && value != "" {
		paths = append(paths, []string{"workspaces", value}...)
	}
	for _, x := range []string{"resource", "name", "subresource", "subresourcename"} {
		if value, found := a.HasParameter(x); found {
			paths = append(paths, value)
		}
	}

	baseuri := strings.TrimPrefix(types.APIVersion, "/")

	// @step: we add the path elements and the queries together
	uri := strings.Join(append([]string{baseuri}, paths...), "/")
	if len(a.queryparams) > 0 {
		uri = fmt.Sprintf("%s?%s", uri, a.queryparams.Encode())
	}

	return uri, nil
}

// makeEndpointURL is responsible for generating the url from a template
func (a *apiClient) makeEndpointURL() (string, error) {
	uri := strings.TrimPrefix(types.APIVersion, "/") + a.endpoint

	// @step: we add the path elements and the queries together
	for param, value := range a.parameters {
		uri = strings.ReplaceAll(uri, "{"+param+"}", value)
	}

	// @step: add the query params if any to the url
	if len(a.queryparams) > 0 {
		uri = fmt.Sprintf("%s?%s", uri, a.queryparams.Encode())
	}

	return uri, nil
}

// makeRequest is responsible for preparing and handling the http request
func (a *apiClient) makeRequest(method, url string) (*http.Response, error) {
	// @step: do we have any thing to encode?
	payload, err := a.makePayload()
	if err != nil {
		return nil, err
	}

	// @step: construct the http request
	request, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(ClientVersionHeader, version.Release)

	// @step: add the authentication from profile
	if err := a.AddAuthorization(request); err != nil {
		return nil, err
	}

	return a.hc.Do(request)
}

// Authorization allows you to override the authorization
func (a *apiClient) Authorization(h string) RestInterface {
	a.authtoken = h

	return a
}

// Unauthenticated indicates no need to add authorization
func (a *apiClient) Unauthenticated() RestInterface {
	a.unauthenticated = true

	return a
}

// AddAuthorization is responsible for adding the authorization headers
func (a *apiClient) AddAuthorization(req *http.Request) error {
	if a.unauthenticated {
		return nil
	}

	if a.authtoken != "" {
		req.Header.Set("Authorization", "Bearer "+a.authtoken)

		return nil
	}

	auth := a.cfg.AuthInfos[a.Profile()]

	switch {
	case auth == nil:
		return NewProfileInvalidError("missing authentication profile", a.Profile())

	case auth.OIDC != nil:
		// @step: has the access token expired
		token, err := jwsutils.NewClaimsFromRawToken(auth.OIDC.IDToken)
		if err == nil && token.HasExpired() {
			if err := a.client.RefreshIdentity(); err != nil {
				return err
			}
		}
		req.Header.Set("Authorization", "Bearer "+auth.OIDC.IDToken)

	case auth.Token != nil:
		req.Header.Set("Authorization", "Bearer "+*auth.Token)

	case auth.Identity != nil:

		expired, err := auth.Identity.IsExpired()
		if err != nil {
			return err
		}
		if expired {
			if err := a.client.RefreshIdentity(); err != nil {
				return err
			}
		}

		req.Header.Set("Authorization", "Bearer "+auth.Identity.Token)
	}

	return nil
}

// handleResponse is responsible for handling the http response from api
func (a *apiClient) handleResponse(resp *http.Response) error {
	// @step: if everything is ok, check for a response and return
	code := resp.StatusCode

	common.LogWithoutContext().WithField("code", code).Debug("received the following http response code")

	if code >= http.StatusOK && code <= 299 {
		if err := a.makeResult(resp, a.result); err != nil {
			return err
		}

		return nil
	}

	// @step: we have encountered an error we need read in a APIError or create one
	apiError := &types.Error{}
	apiError.Code = resp.StatusCode
	apiError.Verb = resp.Request.Method
	apiError.URI = resp.Request.RequestURI

	a.decodeError(resp, apiError)

	if apiError.Message == "" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			apiError.Message = "authorization required"
		case http.StatusNotFound:
			apiError.Message = "kind or resource does not exist"
		case http.StatusForbidden:
			apiError.Message = "request denied, check permissions"
		case http.StatusBadRequest:
			apiError.Message = "api responded with invalid request"
		case http.StatusTooManyRequests:
			apiError.Message = "too many requests, please try again"
		case http.StatusServiceUnavailable:
			apiError.Message = "api service is unavailable"
		case http.StatusMethodNotAllowed:
			apiError.Message = "resource does not support method " + apiError.Verb
		case http.StatusConflict:
			apiError.Message = ObjectModifiedError
		default:
			apiError.Message = "invalid response from api server"
		}
	}

	return apiError
}

func (a *apiClient) decodeError(resp *http.Response, apiError *types.Error) {
	if resp.Body != nil {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			vError := &validation.Error{}
			if err := a.makeResult(resp, vError); err != nil {
				common.LogWithoutContext().WithError(err).Debug("response cannot be decoded into a validation error")
				return
			}
			apiError.Message = vError.Error()
			return
		case http.StatusConflict:
			// Two different types of conflict are represented by 409 - a conflict when trying to
			// write an object to k8s and a conflict with a dependency blocking deletion
			if resp.Header.Get(common.ObjectModifiedHeader) == common.LabelValueTrue {
				apiError.Message = ObjectModifiedError
			} else {
				err := &validation.ErrDependencyViolation{}
				if err := a.makeResult(resp, err); err != nil {
					common.LogWithoutContext().WithError(err).Debug("response cannot be decoded into a dependency validation error")
					return
				}
				apiError.Message = err.Error()
			}
			return
		}

		err := a.makeResult(resp, apiError)
		if err != nil && err != io.EOF {
			common.LogWithoutContext().WithError(err).Debug("response cannot be decoded")
		}
	}
}

// makeResult is responsible for reading the resulting payload
func (a *apiClient) makeResult(resp *http.Response, data interface{}) error {
	a.body = &bytes.Buffer{}

	if resp.Body == nil {
		common.LogWithoutContext().Trace("request to api had no response body present")

		return nil
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	a.body.Write(content)

	common.LogWithoutContext().WithField("body", a.body.String()).Trace("we received the following response from api")

	if data == nil || len(content) == 0 {
		common.LogWithoutContext().Trace("no result has been set to save the payload")

		return nil
	}

	return json.NewDecoder(a.Body()).Decode(data)
}

// makePayload is responsible for encoding the payload if any
func (a *apiClient) makePayload() (io.Reader, error) {
	if a.payload == nil {
		return nil, nil
	}
	b := &bytes.Buffer{}

	if err := json.NewEncoder(b).Encode(a.payload); err != nil {
		return nil, err
	}
	common.LogWithoutContext().WithField("payload", b.String()).Trace("using the attached payload for request")

	return b, nil
}

// HasParameter checks if the parameter exists
func (a *apiClient) HasParameter(key string) (string, bool) {
	value, found := a.parameters[key]

	return value, (found && value != "")
}

// Exists check is the resource exists
func (a *apiClient) Exists() (bool, error) {
	if err := a.Get().Error(); err != nil {
		if !IsNotFound(err) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

// Delete performs a delete
func (a *apiClient) Delete() RestInterface {
	return a.handleRequest(http.MethodDelete)
}

// Get performs a get request
func (a *apiClient) Get() RestInterface {
	return a.handleRequest(http.MethodGet)
}

// Create performs a get request
func (a *apiClient) Create() RestInterface {
	return a.handleRequest(http.MethodPost)
}

// Update performs an put request
func (a *apiClient) Update() RestInterface {
	return a.handleRequest(http.MethodPut)
}

// Post performs a post request
func (a *apiClient) Post() RestInterface {
	return a.handleRequest(http.MethodPost)
}

// Stream is used to websocket the response
// func (a *apiClient) Stream(ctx context.Context) (chan []byte, error) {
// 	return a.MakeStreamRequest(ctx, http.MethodGet)
// }

// Parameters defines a list of parameters for the request
func (a *apiClient) Parameters(params ...ParameterFunc) RestInterface {
	for _, fn := range params {
		param, err := fn()
		if err != nil {
			a.ferror = err

			continue
		}
		if param.IsPath {
			a.parameters[param.Name] = param.Value
		} else {
			if a.queryparams == nil {
				a.queryparams = url.Values{}
			}
			a.queryparams.Add(param.Name, param.Value)
		}
	}

	return a
}

// MakeHTTPClient is responsible for creating the http client
func (a *apiClient) MakeHTTPClient(cacert string) *http.Client {
	if cacert == "" {
		return httputils.DefaultHTTPClient
	}

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if cacert != "" {
		if ok := rootCAs.AppendCertsFromPEM([]byte(cacert)); !ok {
			common.LogWithoutContext().Debug("no certs appended, using system certs only")
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: rootCAs},
		},
	}
}

// SubResource adds a subresource to the operation
func (a *apiClient) SubResource(v string) RestInterface {
	a.parameters["subresource"] = v

	return a
}

// SubResourceName adds a subresource to the operation
func (a *apiClient) SubResourceName(v string) RestInterface {
	a.parameters["subresourcename"] = v

	return a
}

// Payload set the payload of the request
func (a *apiClient) Payload(v interface{}) RestInterface {
	a.payload = v

	return a
}

// Result set the object which we should decode into
func (a *apiClient) Result(v interface{}) RestInterface {
	a.result = v

	return a
}

// injectParam is responsible for injecting the parameter
func (a *apiClient) injectParam(key, value string) {
	if value == "" {
		panic(fmt.Errorf("%q path parameter cannot be empty", key))
	}

	a.parameters[key] = value
}

// Name sets the resource name
func (a *apiClient) Name(v string) RestInterface {
	if v == "" {
		return a
	}

	a.injectParam("name", v)

	return a
}

// Resource set the resource kind in the request
func (a *apiClient) Resource(v string) RestInterface {
	a.injectParam("resource", v)

	return a
}

// Workspace set the workspace
func (a *apiClient) Workspace(v corev1.WorkspaceKey) RestInterface {
	a.parameters["workspace"] = string(v)

	return a
}

// Endpoint defines the endpoint to use
func (a *apiClient) Endpoint(v string) RestInterface {
	a.endpoint = v

	return a
}

// Context sets the request context
func (a *apiClient) Context(ctx context.Context) RestInterface {
	a.ctx = ctx

	return a
}

// Error return any error and resets post
func (a *apiClient) Error() error {
	// we need to reset the error
	defer func() {
		a.ferror = nil
	}()

	return a.ferror
}

// Duplicate duplicates the current request
func (a *apiClient) Duplicate() RestInterface {
	n := &apiClient{
		cfg:         a.cfg,
		endpoint:    a.endpoint,
		parameters:  make(map[string]string),
		payload:     a.payload,
		profile:     a.profile,
		queryparams: a.queryparams,
		result:      a.result,
		client:      a.client,
	}
	for k, v := range a.parameters {
		n.parameters[k] = v
	}

	return n
}

// Follow indicates we follow the stream
func (a *apiClient) Follow(v bool) RestInterface {
	a.follow = v

	return a
}

// Do returns both the response and error
func (a *apiClient) Do() (RestInterface, error) {
	return a, a.ferror
}

// GetPayload returns the payload for inspection
func (a *apiClient) GetPayload() interface{} {
	return a.payload
}

// Body returns the body if any
func (a *apiClient) Body() io.Reader {
	return strings.NewReader(a.body.String())
}
