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

package common

import (
	"regexp"
)

const (
	// WorkspaceKeyMinLength is the minimum length of a workspace key
	WorkspaceKeyMinLength = 3
	// WorkspaceKeyMaxLength is the maximum length of a workspace key
	WorkspaceKeyMaxLength = 5

	// WorkspaceOwnerGroup is the owners of the workspace
	WorkspaceOwnerGroup = "owner"
	// WorkspaceMemberGroup is all members of the workspace
	WorkspaceMemberGroup = "member"
)

var (
	// WorkspaceKeyPattern is a regular expression to match workspace keys
	WorkspaceKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]{2,4}$`)

	// DefaultWorkspaceGroups are the groups which exist in all workspaces regardless
	DefaultWorkspaceGroups = map[string]string{
		WorkspaceOwnerGroup:  "Manages users, groups, access policies and roles within the workspace",
		WorkspaceMemberGroup: "Member of the workspace. No workspace management permissions.",

		// The other groups are 'opinion':
		// WorkspaceAdminRole:                "Super-user access to clusters in workspace",
		// WorkspaceEditorRole:               "Super-user access to namespaces in Non-Production. View only access in Production",
		// WorkspaceViewerRole:               "View only access to all clusters in workspace",
		// WorkspaceTroubleshootingRole:      "Access to logs and exec into running containers",
		// WorkspaceSecretsViewerProdRole:    "Access to secrets, and other sensitive K8s configs",
		// WorkspaceSecretsViewerNonProdRole: "Access to secrets in Non-Production, and other sensitive K8s configs",
	}
)
