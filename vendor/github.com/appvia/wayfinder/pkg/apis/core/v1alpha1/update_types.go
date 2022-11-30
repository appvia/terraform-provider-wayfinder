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

// UpdateObject is the interface that all update compatible objects must implement
// +kubebuilder:object:generate=false
type UpdateObject interface {
	Object
	IsAwaitingApproval() bool
	EstimatePercentageComplete() int
	IsComplete() bool
	GetPreRequisite() *UpdateStep
}

// UpdateSpec are the fields required by an update compatible object on the spec
type UpdateSpec struct {
	// PreRequisite is any steps that can block this update
	// +kubebuilder:validation:Optional
	PreRequisite *UpdateStep `json:"preRequisite,omitempty"`
	// AwaitingApproval specifies that the step is blocked until the user confirms
	// No more reconciliation will be attempted until this is set to false
	// +kubebuilder:validation:Optional
	AwaitingApproval bool `json:"awaitingApproval,omitempty"`
	// Next is the single next steps after this update
	// Not required if there are no further steps
	// Provided as a convenience to allow for a UI to show the next step
	// - E.g. a final manual intervention step could indicate what will happen when the user confirms
	// +kubebuilder:validation:Optional
	Next *UpdateStep `json:"nextSteps,omitempty"`
}

// UpdateStatus are the status fields required by an update compatible object
type UpdateStatus struct {
	// PreRequisites are the current status of the pre-requisites
	// +kubebuilder:validation:Optional
	PreRequisite UpdateStepStatus `json:"preRequisite,omitempty"`
	// StartTime is the time the update was started
	// Is used to estimate the percentage complete time
	// +kubebuilder:validation:Optional
	StartTime metav1.Time `json:"startTime,omitempty"`
	// EstimatedPercentageComplete is the estimated percentage complete of the update
	// - Based on the time from StartTime and a test of actual updates
	// - not optional, will be 0 if not started
	EstimatedPercentageComplete int `json:"estimatedPercentageComplete"`
}

// UpdateSteps is a list of update steps
// envisaged to be present on a rollout plan
type UpdateSteps []UpdateStep

// UpdateStep is the specification of a step in an update plan or a pre-requisite
type UpdateStep struct {
	// Owner is the object (when relevant) that the step is related to
	// +kubebuilder:validation:Optional
	Owner Ownership `json:"owner"`
}

// UpdateStepStatus is the current observed status of an update step
// this is for pre-requisites and next steps
type UpdateStepStatus struct {
	// Owner is the object (when relevant) that the step is related to
	// +kubebuilder:validation:Optional
	Owner Ownership `json:"owner"`
	// Status is the current status of the step
	// +kubebuilder:validation:Required
	Status Status `json:"status"`
	// Error is the error message if the step failed
	// +kubebuilder:validation:MaxLength=32768
	// +kubebuilder:validation:Optional
	Error string `json:"error,omitempty"`
}
