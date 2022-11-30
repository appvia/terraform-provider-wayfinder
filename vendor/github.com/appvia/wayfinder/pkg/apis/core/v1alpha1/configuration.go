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

type ConfigurationFromSourceList []ConfigurationFromSource

type ConfigurationFromSource struct {
	// Path is the JSON path of the configuration parameter
	// Examples: "field", "map_field.value", "array_field.0", "array_field.0.value"
	// To append a value to an existing array: "array_field.-1"
	// To reference a numeric key on a map: "map_field.:123.value"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path"`
	// SecretKeyRef is a reference to a key in a secret
	// +kubebuilder:validation:Required
	SecretKeyRef *OptionalSecretKeySelector `json:"secretKeyRef"`
}

type OptionalSecretKeySelector struct {
	SecretKeySelector `json:",inline"`
	// Optional controls whether the secret with the given key must exist
	// +kubebuilder:validation:Optional
	Optional bool `json:"optional,omitempty"`
}

type SecretKeySelector struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Name is the namespace of the secret
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace,omitempty"`
	// Key is they data key in the secret
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}
