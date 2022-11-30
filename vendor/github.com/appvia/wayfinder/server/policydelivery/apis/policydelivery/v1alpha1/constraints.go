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
)

// Constraints defines a constraint to assuming access to an AccessRole aka creating an AccessRoleBinding
type Constraints struct {
	// Clusters constrains access to matching clusters
	// +kubebuilder:validation:Optional
	Clusters *ClusterConstraint `json:"clusters,omitempty"`
	// Namespaces constraints access to matching namespaces
	Namespaces *NamespaceConstraint `json:"namespaces,omitempty"`
	// Days constrains access to the days of week
	// +kubebuilder:validation:Optional
	Days *DaysOfWeekConstraint `json:"days,omitempty"`
	// TTL is the length of a session. A user's access will be revoked after this amount of time.
	// +kubebuilder:validation:Optional
	TTL *metav1.Duration `json:"ttl,omitempty"`
	// Networks allows defines one of more networks from which the user
	// assuming the role can come from
	// +kubebuilder:validation:Optional
	Networks *NetworkConstraint `json:"networks,omitempty"`
	// Time provides a time of day constraint when assuming the policy. The assumption
	// must occur within the alloted time frame to assume the role
	// +kubebuilder:validation:Optional
	Time *TimeConstraint `json:"time,omitempty"`
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

// NamespaceConstraint places a constraint around the namespaces on clusters
type NamespaceConstraint struct {
	// Allowed is a collection of namespaces permitted access to the role
	// +kubebuilder:validation:Optional
	Allowed *metav1.LabelSelector `json:"allowed,omitempty"`
	// Denied provides the means to denied one or more namespaces from
	// the role
	Denied *metav1.LabelSelector `json:"denied,omitempty"`
}

// StageConstraint places a further constraint around which clusters can be accessed
type StageConstraint struct {
	// Allowed are the stages the clusters the access is allowed to be assumed on
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Denied are the stages the clusters the access is NOT allowed to be assumed on
	// +kubebuilder:validation:Optional
	Denied []string `json:"denied,omitempty"`
}

// WeekDay
// +kubebuilder:validation:Enum=Monday;Tuesday;Wednesday;Thursday;Friday;Saturday;Sunday
type WeekDay string

// DaysOfWeekConstraint places a constraint on the day of week when the role can be assumed
type DaysOfWeekConstraint struct {
	// Allowed are the days permitted in access
	// These are the values from the Weekday type .String() method
	// +kubebuilder:validation:Optional
	Allowed []WeekDay `json:"allowed,omitempty"`
	// Denied is a collection of days which are not permitted to access
	// the role
	// +kubebuilder:validation:Optional
	Denied []WeekDay `json:"denied,omitempty"`
}

// AllowedContains returns true if the day is in the allowed list
func (d *DaysOfWeekConstraint) AllowedContains(day WeekDay) bool {
	found := false
	if d.Allowed == nil {
		return false
	}
	for _, d := range d.Allowed {
		if d == day {
			found = true
			break
		}
	}
	return found
}

// DeniedContains returns true if the day is in the denied list
func (d *DaysOfWeekConstraint) DeniedContains(day WeekDay) bool {
	found := false
	if d.Denied == nil {
		return false
	}
	for _, d := range d.Denied {
		if d == day {
			found = true
			break
		}
	}
	return found
}

// TimeConstraint provides a control around the time a user can assume
// one of more roles in wayfinder
type TimeConstraint struct {
	// Before indicates nothing before this time can access the AccessRole
	// +kubebuilder:validation:Optional
	AllowedFrom *string `json:"from,omitempty"`
	// After indicate the nothing after this time access the role
	// +kubebuilder:validation:Optional
	AllowedTo *string `json:"to,omitempty"`
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
