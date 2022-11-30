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

import corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"

// GetRolesIAMRequest is the input for getting cloud IAM objects
// Used before a cloud access config exists
type GetRolesIAMRequest struct {
	// Name is the name of the cloud access config to template the roles for
	Name string `json:"name"`
	// Workspace is the workspace to template the roles for
	Workspace corev1.WorkspaceKey `json:"workspace"`
	// Cloud is the cloud provider
	Cloud string `json:"provider"`
	// RoleNameOverrides is a list of custom names to use for cloud IAM objects
	// For AWS, must start with wf- at this time
	RoleNameOverrides map[string]string `json:"roleNameOverrides"`
	// DefaultRegion is the default region for accessing the cloud provider
	DefaultRegion string `json:"defaultRegion"`
	// Identifier is the AWS account ID, Azure subscription ID or GCP project ID
	// is used to specify the scope of the IAM permissions
	Identifier string `json:"identifier"`
	// OrgIdentifier for the CloudAccessResource
	// Required on Azure.
	// Set to the Azure Tenant ID
	OrgIdentifier string `json:"orgIdentifier"`
	// Features is a list of features that the IAM roles should support
	Features []string `json:"features"`
	// Stage is the stage of the cloud access config.
	// Only used for naming the CloudAccessConfig resource
	Stage string `json:"stage"`
}

// CloudAccessConfigAndIAM represents any updates to a cloud access config required and IAM roles
type CloudAccessConfigAndIAM struct {
	// CloudAccessConfig is the cloud access config with updates for roles required for the features requested
	CloudAccessConfig *CloudAccessConfig `json:"cloudAccessConfig"`
	// IAM represents the IAM objects required to provide access for Wayfinder roles
	IAM *IAMRoles `json:"iam"`
}

// IAMRoles is a collection of IAM objects required to provide access for Wayfinder roles
type IAMRoles struct {
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Features []string `json:"features"`

	GCP   *GCPIAM   `json:"gcp,omitempty"`
	AWS   *AWSIAM   `json:"aws,omitempty"`
	Azure *AzureIAM `json:"azure,omitempty"`
}
