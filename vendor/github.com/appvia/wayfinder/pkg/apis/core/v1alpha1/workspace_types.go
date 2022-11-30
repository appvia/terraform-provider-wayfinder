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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These are defined here rather than common to keep the dependency tree in a sane order. Ideally,
// we would define them outside the APIs but it just makes the code messier if we do.

const (
	// AdminWorkspace is the workspace where platform administrative resources live
	// **IMPORTANT: IF THIS VALUE IS CHANGED, ENSURE THAT ui/lib/utils/workspaces.ts IS ALSO UPDATED
	AdminWorkspace WorkspaceKey = "admin"
	// WorkspaceNSPrefix is the prefix given to a workspace key to name the namespace that workspace
	// owns in the management cluster
	// **IMPORTANT: IF THIS VALUE IS CHANGED, ENSURE THAT ui/lib/utils/workspaces.ts IS ALSO UPDATED
	WorkspaceNSPrefix = "ws-"
)

// WorkspaceKey is the unique identifier for a workspace in Wayfinder. Use .Namespace() to convert
// to the right name for the workspace's namespace in the management cluster.
type WorkspaceKey string

// Namespace returns the namespace owned by this workspace in the management cluster.
// **IMPORTANT: IF THIS LOGIC IS CHANGED, ENSURE THAT ui/lib/utils/workspaces.ts IS ALSO UPDATED
func (w WorkspaceKey) Namespace() string {
	if w == "" {
		return ""
	}
	return WorkspaceNSPrefix + string(w)
}

// Owns returns true if this workspace owns the given resource (i.e. it lives in this workspace's
// namespace in the management cluster)
func (w WorkspaceKey) Owns(resource metav1.Object) bool {
	return resource.GetNamespace() == w.Namespace()
}

// Key provides a raw string representation of the workspace key. This should be only very rarely
// needed when we absolutely require a pure string.
// ** WARNING: Prefer refactoring code to take, pass and use corev1.WorkspaceKey instead of calling
// this wherever possible.
func (w WorkspaceKey) Key() string {
	return string(w)
}

// WorkspaceKeys is a set of workspace keys
type WorkspaceKeys []WorkspaceKey

// ToWorkspaceKeys returns a set of WorkspaceKeys from a string list of workspace keys
func ToWorkspaceKeys(keys []string) WorkspaceKeys {
	wsKeys := make(WorkspaceKeys, len(keys))
	for i, k := range keys {
		wsKeys[i] = WorkspaceKey(k)
	}

	return wsKeys
}

// Contains returns true if the given key is contained in this set of keys
func (w WorkspaceKeys) Contains(ws WorkspaceKey) bool {
	for _, x := range w {
		if ws == x {
			return true
		}
	}

	return false
}

// KeyList returns the set of workspace keys as a raw string slice for convenience. This should be
// very rarely needed.
func (w WorkspaceKeys) KeyList() []string {
	keys := make([]string, len(w))
	for i, k := range w {
		keys[i] = string(k)
	}
	return keys
}

// ToWorkspace returns the name of the workspace which owns the specified namespace in the
// management cluster
// **IMPORTANT: IF THIS LOGIC IS CHANGED, ENSURE THAT ui/lib/utils/workspaces.ts IS ALSO UPDATED
func ToWorkspace(namespace string) WorkspaceKey {
	if strings.HasPrefix(namespace, WorkspaceNSPrefix) {
		return WorkspaceKey(namespace[len(WorkspaceNSPrefix):])
	}
	return WorkspaceKey(namespace)
}

// Workspace returns the workspace which owns a given resource
func Workspace(resource metav1.Object) WorkspaceKey {
	return ToWorkspace(resource.GetNamespace())
}
