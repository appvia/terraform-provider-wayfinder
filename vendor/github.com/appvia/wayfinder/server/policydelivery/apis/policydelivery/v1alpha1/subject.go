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

import "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"

// SubjectKind represents types of subject that Wayfinder can use
type SubjectKind string

const (
	// SubjectKindUser is a Wayfinder user account
	SubjectKindUser = "User"
	// SubjectKindRobot is a Wayfinder robot account
	SubjectKindRobot = "Robot"
	// SubjectKindWayfinderGroup is a globally defined Wayfinder group
	SubjectKindWayfinderGroup = "WayfinderGroup"
	// SubjectKindWorkspaceGroup is a group defined in a workspace
	SubjectKindWorkspaceGroup = "WorkspaceGroup"
)

// Subject defines a subject for AccessPolicies / AccessRoleBindings etc.
type Subject struct {
	// Kind of object being referenced.
	// Initially supported values include "User", "Robot", "WayfinderGroup" and "WorkspaceGroup"
	Kind string `json:"kind"`
	// Name of the object being referenced.
	Name string `json:"name"`
	// Workspace is the workspace of the subject - only applicable to kind 'Robot'
	Workspace v1alpha1.WorkspaceKey `json:"workspace,omitempty"`
}
