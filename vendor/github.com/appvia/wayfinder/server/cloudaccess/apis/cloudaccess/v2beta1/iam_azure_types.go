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

package v2beta1

import apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// AzureIAM is a collection of Azure role definitions and role assignments
type AzureIAM struct {
	SubscriptionID string                            `json:"subscriptionID"`
	Features       []string                          `json:"features"`
	IAM            []AzureIdentityAndRoleDefinitions `json:"iam"`
}

// AzureIdentityAndRoleDefinitions is an Azure Identity with a role definition and assignment
type AzureIdentityAndRoleDefinitions struct {
	WayfinderRoleName string                 `json:"wayfinderRoleName"`
	ManagedIdentity   *AzureManagedIdentity  `json:"managedIdentity,omitempty"`
	ServicePrincipal  *AzureServicePrincipal `json:"servicePrincipal,omitempty"`
	RoleDefinition    AzureRoleDefinition    `json:"roleDefinition,omitempty"`
}

type AzureManagedIdentity struct {
	IdentityName string `json:"identityName"`
}

type AzureApplication struct {
	TenantID    string `json:"tenantID"`
	DisplayName string `json:"displayName"`
}

type AzureServicePrincipal struct {
	Name        string           `json:"name"`
	Application AzureApplication `json:"application"`
}

type AzureIdentity struct {
	Name string `json:"name"`
	Spec struct {
		ResourceID string `json:"resourceID"`
		ClientID   string `json:"clientID"`
	} `json:"spec"`
}

type AzureIdentityBinding struct {
	Name string `json:"name"`
	Spec struct {
		Selector      string `json:"selector"`
		AzureIdentity string `json:"azureIdentity"`
	} `json:"spec"`
}

type AzureRoleAssignment struct {
	SubscriptionID     string `json:"subscriptionID"`
	RoleDefinitionName string `json:"roleName"`
	IdentityName       string `json:"identityName"`
	// AssignmentScopes are the scopes to which the role assignment applies.
	AssignmentScopes []string `json:"assignmentScopes"`
}

type AzureRoleDefinition struct {
	RoleName         string        `json:"roleName"`
	Description      string        `json:"description"`
	Permissions      apiextv1.JSON `json:"permissions"`
	AssignableScopes []string      `json:"assignableScopes"`
}

func (r *AzureRoleDefinition) SetPermissions(s string) {
	if s == "" {
		r.Permissions.Raw = nil
	}
	r.Permissions = apiextv1.JSON{Raw: []byte(s)}
}
