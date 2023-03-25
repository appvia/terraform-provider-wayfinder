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

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/client/config"
)

// ObjectKey identifies a Wayfinder object
type ObjectKey struct {
	Workspace corev1.WorkspaceKey
	Name      string
}

// ObjectKeyFromObject returns the ObjectKey given a runtime.Object.
func ObjectKeyFromObject(obj Object) ObjectKey {
	return ObjectKey{Workspace: corev1.Workspace(obj), Name: obj.GetName()}
}

// Object is the basic interface implemented by all Wayfinder objects
type Object corev1.Object

// ObjectList is the basic list interface implemented by all Wayfinder list objects
type ObjectList corev1.ObjectList

// Reader knows how to read and list Wayfinder objects.
type Reader interface {
	// Get retrieves an obj for the given object key from the Kubernetes Cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.
	Get(ctx context.Context, key ObjectKey, obj Object) error

	// List retrieves list of objects for a given namespace and list options. On a
	// successful call, Items field in the list will be populated with the
	// result returned from the server.
	List(ctx context.Context, list ObjectList, opts ...ListOption) error
}

// Writer knows how to create, delete, and update Kubernetes objects.
type Writer interface {
	// Create saves the object obj in the Wayfinder API.
	Create(ctx context.Context, obj Object, opts ...CreateOption) error

	// Delete deletes the given obj from Wayfinder API.
	Delete(ctx context.Context, obj Object, opts ...DeleteOption) error

	// Update updates the given obj in the Wayfinder API. obj must be a
	// struct pointer so that obj can be updated with the content returned by the Server.
	Update(ctx context.Context, obj Object, opts ...UpdateOption) error
}

// WFClient provides a simple client to Wayfinder, inspired by controller-manager's client.Client
type WFClient interface {
	Reader
	Writer
}

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
const ObjectModifiedError = "the object has been modified, please try again"

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
	// HasParameter checks if the parameter is set
	HasParameter(string) (string, bool)
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
	// Stream(context.Context) (chan []byte, error)
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
