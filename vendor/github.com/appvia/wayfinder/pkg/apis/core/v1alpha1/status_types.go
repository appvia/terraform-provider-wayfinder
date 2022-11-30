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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status is the status of a thing
type Status string

const (
	// DeletingStatus indicates we are deleting the resource
	DeletingStatus Status = "Deleting"
	// DeleteErrorStatus indicates an error has occurred while attempting to delete the resource
	DeleteErrorStatus Status = "DeleteError"
	// DeletedStatus indicates a deleted entity
	DeletedStatus Status = "Deleted"
	// DeleteFailedStatus indicates that deleting the entity failed
	DeleteFailedStatus Status = "DeleteFailed"
	// ErrorStatus indicates that a recoverable error happened
	ErrorStatus Status = "Error"
	// PendingStatus indicate we are waiting
	PendingStatus Status = "Pending"
	// SuccessStatus is a successful resource
	SuccessStatus Status = "Success"
	// FailureStatus indicates the resource has failed for one or more reasons
	FailureStatus Status = "Failure"
	// WarningStatus indicates are warning
	WarningStatus Status = "Warning"
	// Unknown is an unknown status
	Unknown Status = "Unknown"
	// EmptyStatus indicates an empty status
	EmptyStatus Status = ""
	// CreatingStatus indicate we are creating a resource
	CreatingStatus Status = "Creating"
	// UpdatingStatus indicate we are creating a resource
	UpdatingStatus Status = "Updating"
	// ActionRequiredStatus indicates that user action is required to remediate the current state
	// of a resource, e.g. a spec value is wrong or some external action needs to be taken
	ActionRequiredStatus Status = "ActionRequired"
)

func (s Status) IsSuccess() bool {
	return s == SuccessStatus
}

func (s Status) IsFailed() bool {
	return s == FailureStatus || s == DeleteFailedStatus
}

func (s Status) IsError() bool {
	return s.IsFailed() || s == ErrorStatus || s == DeleteErrorStatus
}

func (s Status) IsDeleting() bool {
	return s == DeletingStatus || s == DeletedStatus || s == DeleteFailedStatus || s == DeleteErrorStatus
}

func (s Status) OneOf(statuses ...Status) bool {
	for _, status := range statuses {
		if status == s {
			return true
		}
	}
	return false
}

// +k8s:openapi-gen=true
type CommonStatus struct {
	// Status is the overall status of the resource. This will shortly become required, hence no
	// omit empty here.
	// +kubebuilder:validation:Optional
	Status Status `json:"status"`
	// Message is a description of the current status
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
	// Detail is any additional human-readable detail to understand the current status, for example,
	// the full underlying error which caused an issue
	// +optional
	Detail string `json:"detail,omitempty"`
	// Conditions represents the observations of the resource's current state.
	// +kubebuilder:validation:Type=array
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:Optional
	Conditions Conditions `json:"conditions,omitempty"`
	// LastReconcile describes the generation and time of the last reconciliation
	// +kubebuilder:validation:Optional
	LastReconcile *LastReconcileStatus `json:"lastReconcile,omitempty"`
	// LastSuccess descibes the generation and time of the last reconciliation which resulted in
	// a Success status
	// +kubebuilder:validation:Optional
	LastSuccess *LastReconcileStatus `json:"lastSuccess,omitempty"`
	// CloudResourcesCreated indicates that at some point, this resource has successfully created
	// one or more cloud resources. This is used when deleting to decide whether to fail or ignore
	// if a related cloud account is inaccessible.
	CloudResourcesCreated bool `json:"cloudResourcesCreated,omitempty"`
	// ObsoleteResources contains a list of resources that are marked for deletion
	// +kubebuilder:validation:Optional
	ObsoleteResources ObsoleteResourceList `json:"obsoleteResources,omitempty"`
}

// Error returns an aggregated list of the errors from conditions or the status.message
func (s CommonStatus) Error() error {
	if err := s.Conditions.Error(); err != nil {
		return err
	}

	return errors.New(s.Message)
}

// +k8s:openapi-gen=true
type LastReconcileStatus struct {
	// Time is the last time the resource was reconciled
	// +kubebuilder:validation:Optional
	Time metav1.Time `json:"time"`
	// Generation is the generation reconciled on the last reconciliation
	// +kubebuilder:validation:Optional
	Generation int64 `json:"generation"`
}

// GetStatus describes the status of this resource
func (s *CommonStatus) GetStatus() (status Status, message string) {
	return s.Status, s.Message
}

// SetStatus sets the overall status of this resource
func (s *CommonStatus) SetStatus(status Status) {
	s.Status = status
}

// GetCommonStatus returns the standard Wayfinder common status information for the resource
func (s *CommonStatus) GetCommonStatus() *CommonStatus {
	return s
}

// CommonStatusAware is implemented by any Wayfinder resource which has the standard Wayfinder common status
// implementation
// +kubebuilder:object:generate=false
type CommonStatusAware interface {
	GetCommonStatus() *CommonStatus
}
