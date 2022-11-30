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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// Constraints defines a constriant to assuming a role
type Constraints struct {
	// Clusters provides a constriant around the cluster which can be assumed
	// +kubebuilder:validation:Optional
	Clusters *ClusterConstraint `json:"clusters,omitempty"`
	// Days provides a constriant around the day of week a role can be assumed
	// +kubebuilder:validation:Optional
	Days *DaysOfWeekConstraint `json:"days,omitempty"`
	// Expiration provides control over the length of a session
	// +kubebuilder:validation:Optional
	Expiration *ExpirationConstraint `json:"expiration,omitempty"`
	// Parameters provides a generic constraint around the requirement inputs
	// into a role
	Parameters []*InputConstraint `json:"parameters,omitempty"`
	// Namespaces provides a constriant around the namespaces which can be assumed
	// into from the role
	Namespaces *NamespaceConstraint `json:"namespaces,omitempty"`
	// Networks allows defines one of more networks from which the user
	// assuming the role can come from
	// +kubebuilder:validation:Optional
	Networks *NetworkConstraint `json:"networks,omitempty"`
	// Roles is a collection of subjects roles which are permitted access
	// to the role
	// +kubebuilder:validation:Optional
	Roles *RolesConstraint `json:"roles,omitempty"`
	// Scopes is a collection of subjects scopes which are permitted
	// access to the role
	// +kubebuilder:validation:Optional
	Scopes *ScopesConstraint `json:"scopes,omitempty"`
	// Subjects is a collection of subjects which are permitted
	// access to the role
	// +kubebuilder:validation:Optional
	Subjects *SubjectsConstraint `json:"subjects,omitempty"`
	// Time provides a time constraint when assuming the policy. The assumption
	// must occur within the alloted time frame to assume the role
	// +kubebuilder:validation:Optional
	Time *TimeConstraint `json:"time,omitempty"`
}

// InputConstraint provides a constriant around an input parameter
type InputConstraint struct {
	// Name is the name of the parameter which maps onto the parameters
	// requirement in the plan
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Resource provides an optional resource definitions to lookup
	// and guard against
	// +kubebuilder:validation:Optional
	Resource *corev1alpha1.Ownership `json:"resource,omitempty"`
	// Allowed is used to dictate the permitted values - when used in conbination
	// with a resource lookup - the values of the parameter
	Allowed *ResourceSelector `json:"allowed,omitempty"`
	// Denied provides a filter around the permitted values as in input
	// to the role
	Denied *ResourceSelector `json:"denied,omitempty"`
}

// ExpirationConstraint define the constraint around the session
type ExpirationConstraint struct {
	// Max is the maximum length of a session the user can assume in the wayfinder
	// +kubebuilder:validation:Optional
	Max *metav1.Duration `json:"max,omitempty"`
}

// ResourceSelector is used to filter on the labels of a resource
type ResourceSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
	// map is equivalent to an element of matchExpressions, whose key field is "key", the
	// operator is "In", and the values array contains only "value". The requirements are ANDed.
	// +kubebuilder:validation:Optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
	// MatchExpressions is a list of label selector requirements. The requirements are ANDed.
	// +kubebuilder:validation:Optional
	MatchExpressions []metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`
	// Values is provides a list of expected values which are permitted
	// +kubebuilder:validation:Optional
	Values []string `json:"values,omitempty"`
}

// HasLabelFilters checks if we have label selectors
func (r *ResourceSelector) HasLabelFilters() bool {
	return r.HasMatchLabels() || r.HasMatchExpressions()
}

// HasMatchExpressions checks if we have match expressions
func (r *ResourceSelector) HasMatchExpressions() bool {
	return len(r.MatchExpressions) > 0
}

// HasMatchLabels checks if we have matching labels
func (r *ResourceSelector) HasMatchLabels() bool {
	return len(r.MatchLabels) > 0
}

// ClusterConstraint places a constraint around the cluster
type ClusterConstraint struct {
	// Allowed is a collection of clusters permitted access on the role
	// +kubebuilder:validation:Optional
	Allowed *metav1.LabelSelector `json:"allowed,omitempty"`
	// Denied provides the means to denied one or more clusters from
	// the role
	Denied *metav1.LabelSelector `json:"denied,omitempty"`
}

// NamespaceConstraint places a constraint around the cluster
type NamespaceConstraint struct {
	// Allowed is a collection of namespaces permitted access on the role
	// +kubebuilder:validation:Optional
	Allowed *metav1.LabelSelector `json:"allowed,omitempty"`
	// Denied provides the means to denied one or more namespaces from
	// the role
	Denied *metav1.LabelSelector `json:"denied,omitempty"`
}

// ScopesConstraint places a constraint around the scopes
type ScopesConstraint struct {
	// Allowed are the subject scopes permitted to access the role
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied are the workspace roles which are not permitted to access the role
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}

// RolesConstraint places a constraint around the workspace roles
type RolesConstraint struct {
	// Allowed are the workspace roles permitted to access to the role
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied are the workspace roles which are not permitted to access the role
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}

// SubjectsConstraint places a constraint around the subjects
type SubjectsConstraint struct {
	// Allowed are the days permitted in access i.e Mon,Tues,Wed,Thu,Fri,Sat or Sun
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied is a collection of days which are not permitted to access
	// the role
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}

// DaysOfWeekConstraint places a constraint on the day of week from which
// the role can be assumed
type DaysOfWeekConstraint struct {
	// Allowed are the days permitted in access i.e Mon,Tues,Wed,Thu,Fri,Sat or Sun
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied is a collection of days which are not permitted to access
	// the role
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}

// TimeConstraint provides a control around the time a user can assume
// one of more roles in wayfinder
type TimeConstraint struct {
	// Before indicates nothing before this time can access the role
	// +kubebuilder:validation:Optional
	Before *string `json:"before,omitempty"`
	// After indicate the nothing after this time access the role
	// +kubebuilder:validation:Optional
	After *string `json:"after,omitempty"`
}

// NetworkConstraint provides a collection of network ranges which
// the user can come from
type NetworkConstraint struct {
	// Allowed is a collection of network which they must originate from
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied is a collection of networks cidrs which will be denied
	// regardless
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}
