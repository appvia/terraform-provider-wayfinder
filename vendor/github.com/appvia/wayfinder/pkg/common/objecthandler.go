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

package common

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// ObjectScopeWorkspace means the object will be used in a workspace scope
	ObjectScopeWorkspace int8 = 1 << iota
	// ObjectScopeGlobal means the object can be used in a global scope
	ObjectScopeGlobal
)

const (
	// ObjectAPIRouteList enables the list endpoint
	ObjectAPIRouteList int = 1 << iota
	// ObjectAPIRouteHas enables the has endpoint
	ObjectAPIRouteHas
	// ObjectAPIRouteGet enables the get endpoint
	ObjectAPIRouteGet
	// ObjectAPIRouteCreate enables the create endpoint
	ObjectAPIRouteCreate
	// ObjectAPIRouteUpdate enables the update endpoint
	ObjectAPIRouteUpdate
	// ObjectAPIRouteDelete enables the delete endpoint
	ObjectAPIRouteDelete
	// ObjectAPIRouteListAll enables the list all endpoint for a workspace-scoped resource
	ObjectAPIRouteListAll
)

const (
	// ObjectAPIRoutesRead contains all read endpoints
	ObjectAPIRoutesRead = ObjectAPIRouteList + ObjectAPIRouteHas + ObjectAPIRouteGet + ObjectAPIRouteListAll
	// ObjectAPIRoutesWrite contains all write endpoints
	ObjectAPIRoutesWrite = ObjectAPIRouteCreate + ObjectAPIRouteUpdate + ObjectAPIRouteDelete
	// ObjectAPIRoutesReadWrite contains all read/write endpoints
	ObjectAPIRoutesReadWrite = ObjectAPIRoutesRead + ObjectAPIRoutesWrite
	// ObjectAPIRoutesReadDelete contains all read methods and the ability to delete but not create
	ObjectAPIRoutesReadDelete = ObjectAPIRoutesRead + ObjectAPIRouteDelete
)

// ObjectFactory can create an API object or an API list object
type ObjectFactory interface {
	// CreateObject returns with a pointer to a new object instance
	CreateObject() corev1alpha1.Object
	// CreateListObject returns with a pointer to a new list object instance
	CreateListObject() corev1alpha1.ObjectList
}

type ObjectDescriptor interface {
	// FeatureGate returns the name of the feature gate which enables the handler
	// Can be left empty in which case the handler is always enabled
	FeatureGate() string
	// Subject returns with the description of the object
	Subject() string
	// APIRoutes returns with the enabled API routes
	// You can compose a custom set of routes using the ObjectAPIRoute* flag constants
	APIRoutes() int
	// Scope returns with the API scope flags (workspace, global or both)
	Scope() int8
	// ListFilters returns the available list filters
	ListFilters() []ObjectListFilter
	// Allocatable return true if this is a resource that supports allocation across workspace
	// boundaries.
	//
	// WARNING: The CRD returned by CreateObject must implement corev1.Allocatable if this
	// returns true.
	Allocatable() bool
	// Tags are used for grouping API objects (e.g. in OpenAPI)
	Tags() []string
	// ClusterScoped returns true if this resource is implemented as a cluster-scoped CRD. If this is
	// true, Scope *must* be only Global.
	ClusterScoped() bool
}

type ObjectListFilter struct {
	ParamName   string
	IsList      bool
	Description string
	Filter      func(value interface{}) func(client.Object) bool
}

type ObjectReader interface {
	// List returns the existing objects filtered by the filters defined in list options
	List(context.Context, corev1alpha1.ObjectList, ...client.ListOption) error
	// Has checks if the given object exists
	Has(context.Context, client.Object) (bool, error)
	// Get returns the object
	Get(context.Context, client.Object) error
}

type ObjectWriter interface {
	// Create is responsible for creating an object
	Create(context.Context, client.Object, ...client.CreateOption) error
	// Update is responsible for updating an object
	Update(context.Context, client.Object, ...client.UpdateOption) error
	// Delete is responsible for deleting an object
	Delete(context.Context, client.Object, ...client.DeleteOption) error
	// DeleteDependencyCheck will do a dry-run of the delete to check for any dependencies that will block
	// deletion. Convenience wrapper around using Delete() with dry run et al for controllers.
	DeleteDependencyCheck(context.Context, client.Object) error
}

type ObjectClient interface {
	ObjectReader
	ObjectWriter
}

type ObjectHandler interface {
	ObjectFactory
	ObjectDescriptor
	ObjectClient
}

// ObjectHandlerFactory is the signature for an object handler that can be registered
type ObjectHandlerFactory func(func() client.Client) ObjectHandler

// ObjectValidator provides the interface for the webhook validation code
type ObjectValidator interface {
	// CheckCreate verifies whether the object can be created
	CheckCreate(ctx context.Context, o client.Object, options ...client.CreateOption) error
	// CheckUpdate verifies whether the object can be updated
	CheckUpdate(ctx context.Context, before, after client.Object, options ...client.UpdateOption) error
	// CheckDelete verifies whether the object can be deleted
	CheckDelete(ctx context.Context, o client.Object, options ...client.DeleteOption) error
	// UseWebhooks should return true, if the webhook server will handle validation/admission webhooks
	// Requires code change in the API definition and in the controller before you can set this to true
	UseWebhooks() bool
}

// ObjectMutator is an optional interface which can be implemented by the object handler
// If implemented and webhooks are enabled, the Mutate function will be called at the end of the admission request
type ObjectMutator interface {
	Mutate(_ context.Context, _ client.Object, dryRun bool) error
}

// NoopObjectValidator creates a new object validator which does no validation
func NoopObjectValidator() ObjectValidator {
	return noopObjectValidator{}
}

type noopObjectValidator struct {
}

func (v noopObjectValidator) CheckCreate(context.Context, client.Object, ...client.CreateOption) error {
	return nil
}

func (v noopObjectValidator) CheckUpdate(context.Context, client.Object, client.Object, ...client.UpdateOption) error {
	return nil
}

func (v noopObjectValidator) CheckDelete(context.Context, client.Object, ...client.DeleteOption) error {
	return nil
}

func (v noopObjectValidator) UseWebhooks() bool {
	return false
}
