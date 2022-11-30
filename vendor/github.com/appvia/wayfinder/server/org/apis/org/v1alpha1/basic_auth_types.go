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

// UpdateBasicAuthIdentity defines the desired state of an update
type UpdateBasicAuthIdentity struct {
	// Password is a password associated to the user
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Password string `json:"password"`
	// Username is the user you are update the credential for
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Username string `json:"username"`
}

// UpdateIDPIdentity defines the desired state of an update
type UpdateIDPIdentity struct {
	// IDToken is the identity token from the provider
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	IDToken string
}
