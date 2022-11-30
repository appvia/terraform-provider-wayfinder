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

import (
	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// SubjectKind represents types of subject that Wayfinder can use
// +kubebuilder:validation:Enum=User;PlatformAccessToken;WorkspaceAccessToken;WayfinderGroup;WorkspaceGroup
type SubjectKind string

const (
	// SubjectKindUser is a Wayfinder user account
	SubjectKindUser SubjectKind = "User"
	// SubjectKindPlaformAccessToken is a Wayfinder-scoped access token
	SubjectKindPlaformAccessToken SubjectKind = "PlatformAccessToken"
	// SubjectKindWorkspaceAccessToken is a Workspace-scoped access token
	SubjectKindWorkspaceAccessToken SubjectKind = "WorkspaceAccessToken"
	// SubjectKindWayfinderGroup is a globally defined Wayfinder group
	SubjectKindWayfinderGroup SubjectKind = "WayfinderGroup"
	// SubjectKindWorkspaceGroup is a group defined in a workspace
	SubjectKindWorkspaceGroup SubjectKind = "WorkspaceGroup"
)

// Subject defines a subject for group memberships and role bindings.
type Subject struct {
	// Kind of subject being referenced.
	Kind SubjectKind `json:"kind"`
	// Name of the object being referenced.
	Name string `json:"name"`
	// Workspace is the workspace of the subject - only applicable to kind 'WorkspaceAccessToken'
	Workspace corev1.WorkspaceKey `json:"workspace,omitempty"`
}

// Equals returns true if the subjects both refer to the same underlying subject
func (s Subject) Equals(other Subject) bool {
	return s.Kind == other.Kind && s.Name == other.Name && s.Workspace == other.Workspace
}

// // Subjective is an interface which can be implemented by the underlying kinds which can be
// // represented as a subject
// type Subjective interface {
// 	// ToSubject provides a subject reference to this thing
// 	ToSubject() Subject
// }
