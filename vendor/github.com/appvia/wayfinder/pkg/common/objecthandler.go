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

	"k8s.io/apimachinery/pkg/runtime/schema"
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
	// ObjectScopeDual is an object that is available at both platform and workspace scope
	ObjectScopeDual = ObjectScopeWorkspace + ObjectScopeGlobal
)

type ObjectListFilter struct {
	ParamName   string
	IsList      bool
	Description string
	// DefaultValue will cause this filter to be ALWAYS used; if the parameter is not specified,
	// this default value will be passed to the filter function. Only
	DefaultValue string
	Filter       func(value interface{}) func(client.Object) bool
}

type ObjectClient interface {
	// List returns the existing objects filtered by the filters defined in list options
	List(context.Context, corev1alpha1.ObjectList, ...client.ListOption) error
	// Get returns the object
	Get(context.Context, client.Object) error
	// Create is responsible for creating an object
	Create(context.Context, client.Object, ...client.PatchOption) error
	// Update is responsible for updating an object
	Update(context.Context, client.Object, ...client.PatchOption) error
	// Delete is responsible for deleting an object
	Delete(context.Context, client.Object, ...client.DeleteOption) error
}

type ObjectHandler interface {
	// ObjectKind must return the GVK of the Wayfinder API type managed by this handler
	ObjectKind() schema.GroupVersionKind
	// CreateObject returns with a pointer to a new object instance
	CreateObject() corev1alpha1.Object
	// Scope returns with the API scope flags (workspace, global or dual)
	Scope() int8
	// Allocatable return true if this is a resource that supports allocation across workspace
	// boundaries.
	//
	// WARNING: The CRD returned by CreateObject must implement corev1.Allocatable if this
	// returns true.
	Allocatable() bool
	// Tags are used for grouping API objects (e.g. in OpenAPI)
	Tag() string

	// ListFilters returns a list of filters to provide as optional query parameters on the list API
	ListFilters() []ObjectListFilter

	// Mutate is called by the mutation webhook for this type
	Mutate(ctx context.Context, obj client.Object, oldObj client.Object, dryRun bool) error

	// CheckCreate must perform any validation required for this type on creation. This is called
	// by the validation webhook for this type when an object is first created. Mutate will be
	// called before CheckCreate, so it is OK to depend on mutations being represented in the
	// provided object.
	CheckCreate(ctx context.Context, o client.Object, dryRun bool) error

	// CheckUpdate must perform any validation required for this type on update. This is called
	// by the validation webhook for this type when an object is updated. Mutate will be called
	// before CheckUpdate, so it is OK to depend on mutations being represented in the provided
	// object.
	CheckUpdate(ctx context.Context, before, after client.Object, dryRun bool) error

	// CheckDelete must perform any pre-delete validation required for this object. This is called
	// by the validation webhook for this type when deletion of the object is requested. We don't
	// typically mutate on delete, but if we did, this would happen after that mutation.
	CheckDelete(ctx context.Context, o client.Object, dryRun bool) error
}

// ObjectHandlerFactory is the signature for an object handler that can be registered
type ObjectHandlerFactory func(func() client.Client) ObjectHandler
