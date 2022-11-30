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

// obsoleteResourceKind defines a supported obsolete resource kind
type obsoleteResourceKind string

const (
	IAMRoleKind obsoleteResourceKind = "IAMRole"
)

// String converts a resource kind to a string value
func (p obsoleteResourceKind) String() string {
	return string(p)
}

type ObsoleteResourceList []ObsoleteResource

// ObsoleteResource is a resource that is marked for deletion
// +k8s:openapi-gen=true
type ObsoleteResource struct {
	// Kind is the kind of the resource, eg. IAMRole
	Kind obsoleteResourceKind `json:"kind"`
	// Name is the name of the resource, eg. my-iam-role
	Name string `json:"name"`
}
