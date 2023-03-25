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
	"strings"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// GetAllocationPrefixes returns a map of common allocation prefixes, which
// will hopefully be deprecated once we relook at allocations as a whole
func GetAllocationPrefixes() map[string]string {
	return map[string]string{
		"AKS": "cloudaccount-azure",
		"EKS": "cloudaccount-aws",
		"GKE": "cloudaccount-gcp",
	}
}

// AllocationChecker represents a function which checks that a workspace is allowed to use thing
type AllocationChecker func(ctx context.Context, workspace string, thing corev1alpha1.Ownership) (bool, error)

// GetUniqueName returns the workspace-qualified name for an object if it lives in a different
// workspace, or the regular object name if it lives in the provided workspace. This gives a
// consistent naming strategy for allocated objects.
//
// If an empty workspace is provided, this will always return the workspace-qualified version of the
// name
func GetUniqueName(object corev1alpha1.Object, ws corev1alpha1.WorkspaceKey) string {
	if ws == "" || object.GetNamespace() != ws.Namespace() {
		// get the cluster workspace key from the namespace of the cluster
		return corev1alpha1.Workspace(object).Key() + "." + object.GetName()
	} else {
		return object.GetName()
	}
}

// SetUniqueNameAnnotation ensures that the passed-in object has the unique name set appropriately
// for the context of the provided workspace
func SetUniqueNameAnnotation(object corev1alpha1.Object, ws corev1alpha1.WorkspaceKey) {
	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[AnnotationUniqueName] = GetUniqueName(object, ws)
	object.SetAnnotations(annotations)
}

// ParseUniqueName returns the workspace and object name from the unique name
// If the unique name is not in the format workspace.clustername then the requesting workspace is
// returned with the unique name
func ParseUniqueName(uniqueName string, requestingWorkspace corev1alpha1.WorkspaceKey) (corev1alpha1.WorkspaceKey, string) {
	// Split the displayname into workspace and cluster name
	split := strings.Split(uniqueName, ".")
	if len(split) == 2 {
		return corev1alpha1.WorkspaceKey(split[0]), split[1]
	}
	return requestingWorkspace, uniqueName
}
