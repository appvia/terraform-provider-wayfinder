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

// GCPIAM is a collection of GCP roles
type GCPIAM struct {
	ProjectID string                         `json:"projectID"`
	Features  []string                       `json:"features"`
	IAM       []GCPServiceAccountAndBindings `json:"iam"`
}

type GCPServiceAccountAndBindings struct {
	WayfinderRoleName      string                   `json:"wayfinderRoleName"`
	ServiceAccount         GCPServiceAccount        `json:"serviceAccount"`
	ServiceAccountBindings GCPServiceAccountBinding `json:"bindings"`
	ProjectBindings        GCPProjectBindings       `json:"projectBindings"`
	OrgBindings            GCPOrgBindings           `json:"orgBindings"`
}

type GCPServiceAccount struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type Binding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

type GCPServiceAccountBinding struct {
	Email    string    `json:"displayName"`
	Bindings []Binding `json:"binding"`
}

type GCPProjectBindings struct {
	ProjectID string    `json:"projectID"`
	Bindings  []Binding `json:"binding"`
}

type GCPOrgBindings struct {
	OrgID    string    `json:"orgID"`
	Bindings []Binding `json:"binding"`
}
