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

// ResourceAllocationType represents the possible types of resource allocation
type ResourceAllocationType string

const (
	// ResourceAllocationWorkspaces indicates that the resource can be used by a specified set of
	// workspaces
	ResourceAllocationWorkspaces ResourceAllocationType = "workspaces"
	// ResourceAllocationAll indicates that the resource can be used by all workspaces
	ResourceAllocationAll ResourceAllocationType = "all"
	// ResourceAllocationNone indicates that the resource can only be used by the workspace that
	// owns it
	ResourceAllocationNone ResourceAllocationType = "none"
)

// ResourceAllocation describes who is allowed to use a resource across workspace boundaries.
type ResourceAllocation struct {
	// Type controls which workspaces can use this resource . If 'none', this resource cannot be
	// used by workspaces other than the one the resource exists in. 'all' allows it to be used by
	// all workspaces, and 'workspaces' indicates it can be used by the workspaces listed in the
	// workspaces property.
	// +kubebuilder:validation:Enum=all;workspaces;none
	// +kubebuilder:validation:Required
	Type ResourceAllocationType `json:"type"`
	// Workspaces indicates which workspaces can use this resource. Ignored unless type is set to
	// 'workspaces'.
	// +kubebuilder:validation:Optional
	Workspaces WorkspaceKeys `json:"workspaces,omitempty" type:"[]string"`
}

func (r ResourceAllocation) Equal(a ResourceAllocation) bool {
	if r.Type != a.Type {
		return false
	}
	if len(r.Workspaces) != len(a.Workspaces) {
		return false
	}
	// Now look through all the workspaces and check they are present
	for _, rt := range r.Workspaces {
		found := false
		for _, at := range a.Workspaces {
			if at == rt {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Allowed returns true if the specified workspace is allowed to use this resource according to the
// specified type (and workpace list if relevant).
func (r ResourceAllocation) Allowed(resourceNamespace string, workspace WorkspaceKey) bool {
	// Workspaces are always allowed to use resources defined in their own namespace:
	if resourceNamespace == workspace.Namespace() {
		return true
	}

	// Elsewise, we need to check the resource allocation:
	switch r.Type {
	case ResourceAllocationAll:
		return true
	case ResourceAllocationWorkspaces:
		for _, w := range r.Workspaces {
			if w == workspace {
				return true
			}
		}
	}

	return false
}

// Allocatable must be implemented by CRDs which are allocateable
// +kubebuilder:object:generate=false
type Allocatable interface {
	// AllocatedToWorkspace must return true if the specified workspace is allowed to use this
	// resource, false otherwise.
	AllocatedToWorkspace(ws WorkspaceKey) bool
}
