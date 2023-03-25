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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true
type RoutingStatus struct {
	// RoutingTargetStatuses is that
	// +kubebuilder:validation:Type=array
	Routing RoutingTargetStatuses `json:"routing,omitempty"`
}

type RoutingTargetStatuses []RoutingTargetStatus

// RoutingTargetStatus is the current observed status of a routing action to a target
// +k8s:openapi-gen=true
type RoutingTargetStatus struct {
	// +required
	// +kubebuilder:validation:Required
	Target Ownership `json:"target"`
	// +required
	// +kubebuilder:validation:Required
	Status Status `json:"status"`
	// +optional
	// +kubebuilder:validation:MaxLength=32768
	Error string `json:"error,omitempty"`
	// LastReconcile describes the generation and time of the last reconciliation
	// +kubebuilder:validation:Optional
	LastReconcile *LastReconcileStatus `json:"lastReconcile,omitempty"`
}

func (rs *RoutingStatus) GetStatusFor(target Object) RoutingTargetStatus {
	ownership := GetOwnershipFromObject(target)
	for _, t := range rs.Routing {
		if t.Target.Equals(ownership) {
			return t
		}
	}
	return RoutingTargetStatus{
		Target: ownership,
		Status: EmptyStatus,
	}
}

func (rs *RoutingStatus) SetStatusFor(target Object, status RoutingTargetStatus) bool {
	ownership := GetOwnershipFromObject(target)
	for i := range rs.Routing {
		if rs.Routing[i].Target.Equals(ownership) {
			rs.Routing[i].Status = status.Status
			rs.Routing[i].Error = status.Error
			if status.LastReconcile != nil {
				rs.Routing[i].LastReconcile = &LastReconcileStatus{
					Time:       status.LastReconcile.Time,
					Generation: status.LastReconcile.Generation,
				}
			}
			return false
		}
	}

	rs.Routing = append(rs.Routing, status)
	return true
}

func (rs *RoutingStatus) RemoveStatusesExcluding(targets []Ownership) {
	new := []RoutingTargetStatus{}
	for _, status := range rs.Routing {
		for _, target := range targets {
			if status.Target.Equals(target) {
				new = append(new, status)
			}
		}
	}
	rs.Routing = new
}

// LastReconcileGeneration returns the last generation we attempted to reconcile
func (rts *RoutingTargetStatus) LastReconcileGeneration() int64 {
	if rts.LastReconcile != nil {
		return rts.LastReconcile.Generation
	}
	return 0
}

// LastReconcileTime returns the last time we attempted to reconcile
func (rts *RoutingTargetStatus) LastReconcileTime() metav1.Time {
	if rts.LastReconcile != nil {
		return rts.LastReconcile.Time
	}
	return metav1.NewTime(time.Time{})
}

// RoutingStatusAware is implemented by any Wayfinder resource which has the standard Wayfinder routuing status
// implementation
// +kubebuilder:object:generate=false
type RoutingStatusAware interface {
	GetRoutingStatus() *RoutingStatus
}
