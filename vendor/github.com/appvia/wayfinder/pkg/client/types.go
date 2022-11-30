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
	"context"
	"io"
	"net/http"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/client/config"
)

// Interface is the api client interface
type Interface interface {
	// Request creates a request instance
	Request() RestInterface
	// Config returns the underlying configuration for the client
	Config() *config.Config
	// CurrentProfile returns the current profile
	CurrentProfile() string
	// OverrideProfile allows you set the selected profile
	OverrideProfile(string) Interface
	// RefreshIdentity is used to refresh the identity token of the user
	RefreshIdentity() error
}

// UpdateHandlerFunc is external method when the configuration has been updated
type UpdateHandlerFunc func() error

// ClientVersionHeader is the header used in the client cli
const ClientVersionHeader = "X-Client-Version"

// Plugin provides an interface for plugins
type Plugin interface {
	// WrapTransport provides the entrypoint for the authentication handle
	WrapTransport(http.RoundTripper) http.RoundTripper
}

// RestInterface provides the rest interface
type RestInterface interface {
	// Authorization allows you to override the authorization
	Authorization(string) RestInterface
	// Body returns the body if any
	Body() io.Reader
	// Context sets the request context
	Context(context.Context) RestInterface
	// Create performs a post request
	Create() RestInterface
	// Delete performs a delete
	Delete() RestInterface
	// Do returns the response and error
	Do() (RestInterface, error)
	// Duplicate copes the current request
	Duplicate() RestInterface
	// Endpoint defines the endpoint to use
	Endpoint(string) RestInterface
	// Exists checks if the resource exists
	Exists() (bool, error)
	// Error returns the error if any
	Error() error
	// Follow indicates we follow the stream from the server if any
	Follow(bool) RestInterface
	// Get performs a get request
	Get() RestInterface
	// GetPayload returns the payload for inspection
	GetPayload() interface{}
	// HasParamater checks if the parameter is set
	HasParamater(string) (string, bool)
	// Name sets the resource name
	Name(string) RestInterface
	// Resource set the resource kind in the request
	Resource(string) RestInterface
	// Parameters defines a list of parameters for the request
	Parameters(...ParameterFunc) RestInterface
	// Payload set the payload of the request
	Payload(interface{}) RestInterface
	// Post performs a post request
	Post() RestInterface
	// Result set the object which we should decode into
	Result(interface{}) RestInterface
	// Stream is used to stream from the api server
	Stream(context.Context) (chan []byte, error)
	// SubResource adds a subresource to the operation
	SubResource(string) RestInterface
	// SubResourceName sets the sub-resource name
	SubResourceName(string) RestInterface
	// Workspace set the workspace
	Workspace(corev1.WorkspaceKey) RestInterface
	// Unauthenticated indicates no need to add auth
	Unauthenticated() RestInterface
	// Update performs an put request
	Update() RestInterface
}

// ParameterFunc defines a method for a parameter type
type ParameterFunc func() (Parameter, error)

// Parameter is a param to the raw endpoint
type Parameter struct {
	// IsPath indicates if it's a path or query parameter
	IsPath bool
	// Name is the name of the parameter
	Name string
	// Value is the value of the parameter
	Value string
}
