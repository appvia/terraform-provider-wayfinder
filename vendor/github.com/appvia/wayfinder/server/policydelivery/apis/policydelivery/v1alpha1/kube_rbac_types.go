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
	rbacv1 "k8s.io/api/rbac/v1"
)

// KubeRBACParameters is the parameters for
type KubeRBACParameters struct {
	// Rules is rules
	// Leave blank to specify only a binding is to be created
	// +kubebuilder:validation:Optional
	Rules []rbacv1.PolicyRule `json:"rules"`
	// Subjects / principles who the rules apply to
	// - user
	// - group
	// - serviceaccount
	// Note only a service account can specify the namespace
	// +kubebuilder:validation:Optional
	Subjects []rbacv1.Subject `json:"subjects"`
	// RoleNameOverride is the name of the Role or ClusterRole if different from the clusterpolicy name
	// - useful when creating just a role binding or referencing a clusterwide role
	// - must be specified when ClusterWideRole is set to true
	// +kubebuilder:validation:Optional
	RoleNameOverride *string `json:"roleNameOverride,omitempty"`
	// ClusterScopedRoleOverride specifies the role referenced by the RoleBinding is a ClusterRole
	// - specify true to override the default when targeting a namespaced role / role binding
	// - ignored when targeting a ClusterRole
	// +kubebuilder:validation:Optional
	ClusterScopedRoleOverride *bool `json:"clusterScopedRoleOverride,omitempty"`
	// RoleBindingOverrideName is the name of the RoleBinding or ClusterRoleBinding if different from the clusterpolicy name
	// Leave blank to have it derived from the name of the clusterpolicy name
	// Specify a unique name depending on the subject and role combination
	// - useful when creating just a role binding or referencing a clusterwide role
	// kubebuilder:validation:Optional
	RoleBindingOverrideName *string `json:"roleBindingNameOverride,omitempty"`
}

// HasBinding returns true if the parameters have subjects
func (p KubeRBACParameters) HasBinding() bool {
	return p.Subjects != nil && len(p.Subjects) > 0
}

// HasRole returns true if the parameters specify rules
func (p KubeRBACParameters) HasRole() bool {
	return p.Rules != nil && len(p.Rules) > 0
}
