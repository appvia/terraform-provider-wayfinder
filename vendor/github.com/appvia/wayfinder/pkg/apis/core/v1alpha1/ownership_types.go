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

package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ownership indicates the ownership of a resource
// +k8s:openapi-gen=true
type Ownership struct {
	// Group is the api group
	Group string `json:"group"`
	// Version is the group version
	Version string `json:"version"`
	// Kind is the name of the resource under the group
	Kind string `json:"kind"`
	// Namespace is the location of the object
	Namespace string `json:"namespace"`
	// Name is name of the resource
	Name string `json:"name"`
}

// Workspace returns the workspace that this owns the resource this points to
func (o Ownership) Workspace() WorkspaceKey {
	return ToWorkspace(o.Namespace)
}

// APIVersion returns the api version
func (o Ownership) APIVersion() string {
	return o.Group + "/" + o.Version
}

// IsCloudAccount returns whether this ownership represents a CloudAccount
func (o Ownership) IsCloudAccount() bool {
	return o.Kind == "CloudAccount" &&
		o.Group == "cloudaccess.appvia.io"
}

func GetOwnershipFromObject(obj client.Object) Ownership {
	return Ownership{
		Group:     obj.GetObjectKind().GroupVersionKind().Group,
		Version:   obj.GetObjectKind().GroupVersionKind().Version,
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func (o Ownership) IsSameType(o2 Ownership) bool {
	return strings.EqualFold(o.Group, o2.Group) &&
		strings.EqualFold(o.Version, o2.Version) &&
		strings.EqualFold(o.Kind, o2.Kind) &&
		strings.EqualFold(o.Namespace, o2.Namespace)
}

func (o Ownership) Equals(o2 Ownership) bool {
	return o.IsSameType(o2) && o.Name == o2.Name
}

func (o Ownership) HasGroupVersionKind(gvk schema.GroupVersionKind) bool {
	return strings.EqualFold(gvk.Group, o.Group) && strings.EqualFold(gvk.Version, o.Version) && strings.EqualFold(gvk.Kind, o.Kind)
}

func (o Ownership) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      o.Name,
		Namespace: o.Namespace,
	}
}

func (o Ownership) ToUnstructured() unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(o.GroupVersionKind())
	u.SetNamespace(o.Namespace)
	u.SetName(o.Name)
	return u
}

func (o Ownership) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   o.Group,
		Version: o.Version,
		Kind:    o.Kind,
	}
}

func (o Ownership) Owns(obj client.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()

	switch {
	case gvk.Group != o.Group:
		return false
	case gvk.Version != o.Version:
		return false
	case gvk.Kind != o.Kind:
		return false
	case obj.GetNamespace() != o.Namespace:
		return false
	case obj.GetName() != o.Name:
		return false
	}

	return true
}

// IsOwn returns true if this ownership points to the exact same object as the other ownership
// provided
// NOTE: MDH 2022-04-28 - this was migrated from pkg/wayfinder/helpers.go - I have no idea if this
// really has any value but it's better placed here than there
func (o Ownership) IsOwn(other Ownership) bool {
	fields := map[string]string{
		o.Group:     other.Group,
		o.Version:   other.Version,
		o.Kind:      other.Kind,
		o.Namespace: other.Namespace,
		o.Name:      other.Name,
	}
	for k, v := range fields {
		if k != v {
			return false
		}
	}

	return true
}

// GetGroupVersionKind returns the gvk
func (o Ownership) GetGroupVersionKind() string {
	return fmt.Sprintf("%s/%s/%s", o.Group, o.Version, o.Kind)
}

// String returns a string representation
func (o Ownership) String() string {
	return fmt.Sprintf("%s.%s/%s/%s/%s", o.Kind, o.Group, o.Version, o.Namespace, o.Name)
}
