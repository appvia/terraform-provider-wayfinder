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

import (
	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// Conditions for all cloud access:

	// ConditionAccessible will be set for all accounts to indicate whether Wayfinder has access to the
	// referenced account, including any required access roles being deployed into that account
	ConditionAccessible corev1alpha1.ConditionType = "Accessible"

	// Conditions for managed accounts:

	// ConditionAccountNameAvailable will be set for managed accounts indicating whether the
	// requested account name is available
	ConditionAccountNameAvailable corev1alpha1.ConditionType = "AccountNameAvailable"
	// ConditionOrgAvailable will be set for managed accounts indicating that the referenced
	// organizational cloud access config is ready for use, accessible by Wayfinder, and that the organization
	// is allocated to this workspace
	ConditionOrgAvailable corev1alpha1.ConditionType = "OrganizationAvailable"
	// ConditionAccountProvisioned will be set for managed accounts to indicate whether the account
	// has been provisioned successfully with the cloud provider
	ConditionAccountProvisioned corev1alpha1.ConditionType = "AccountProvisioned"
	// ConditionUserRolesAvailable will be set for managed accounts when all required user roles are
	// deployed and ready to use
	ConditionUserRolesAvailable corev1alpha1.ConditionType = "UserRolesAvailable"

	// Conditions for org accounts:

	// ConditionNamingRulesConfigured will be set for org accounts when naming rules are correctly
	// configured
	ConditionNamingRulesConfigured corev1alpha1.ConditionType = "NamingRulesConfigured"
	// ConditionOrgDeployed will be set for organization accounts to indicate whether the
	// organization-level setup is up to date with the cloud provider
	ConditionOrgDeployed corev1alpha1.ConditionType = "OrganizationDeployed"
)

const (
	// ReasonAccountNameInUse will be used when AccountNameAvailable is false because the required
	// account name is already in use by another workspace.
	ReasonAccountNameInUse = "AccountNameInUse"
)

var OrgConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionAccessible, Name: "Accessible by Wayfinder"},
	{Type: ConditionNamingRulesConfigured, Name: "Naming Rules Configured"},
	{Type: ConditionOrgDeployed, Name: "Organization deployed"},
	{Type: corev1alpha1.ConditionReady, Name: "Cloud account available"},
}

var SharedAccountConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionAccessible, Name: "Accessible by Wayfinder"},
	{Type: corev1alpha1.ConditionReady, Name: "Cloud account available"},
}

var ManagedAccountConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionOrgAvailable, Name: "Cloud organization ready and accessible"},
	{Type: ConditionAccountNameAvailable, Name: "Account name validated"},
	{Type: ConditionAccountProvisioned, Name: "Account provisioned"},
	{Type: ConditionAccessible, Name: "Accessible by Wayfinder"},
	{Type: corev1alpha1.ConditionReady, Name: "Cloud account ready for use"},
	{Type: ConditionUserRolesAvailable, Name: "User roles available"},
}

const (
	// ConditionMetadataAvailable is set by the cloudmeta operator on the cloud access config when the
	// cloud access config is configured for CloudMeta to indicate the status of the metadata download.
	ConditionMetadataAvailable corev1alpha1.ConditionType = "MetadataAvailable"
)

var MetadataConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionMetadataAvailable, Name: "Cloud regions and instance types downloaded"},
}

const (
	// ConditionRolesAutoSetup is set to indicate the progress or completion of the roles required
	// for the cloud access config being automatically provisioned by Wayfinder
	ConditionRolesAutoSetup corev1alpha1.ConditionType = "RolesAutoSetup"
)

var CloudRolesConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionRolesAutoSetup, Name: "Cloud role auto-provision"},
}
