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

	// WorkspaceOwnerRole is the owners of the workspace
	WorkspaceOwnerRole = "owner"
	// WorkspaceMemberRole is a member of the workspace
	WorkspaceMemberRole = "member"

	// WorkspaceAdminRole is the default admin role
	WorkspaceAdminRole = "admin"

	// WorkspaceEditorRole allows editing of kubernetes objects in a workspace in the default policies
	WorkspaceEditorRole = "editor"
	// WorkspaceViewerRole allows viewing of kubernetes objects in a workspace in the default policies
	WorkspaceViewerRole = "viewer"
	// WorkspaceTroubleshootingRole allows pod logs to be inspected in the default policies
	WorkspaceTroubleshootingRole = "troubleshooting"
	// WorkspaceSecretsViewerProdRole members can get secrets from prod in the default policies
	WorkspaceSecretsViewerProdRole = "secretsviewer-prod"
	// WorkspaceSecretsViewerProdRole members can get secrets from prod in the default policies
	WorkspaceSecretsViewerNonProdRole = "secretsviewer-nonprod"
)

var (
	// WorkspaceKeyPattern is a regular expression to match workspace keys
	WorkspaceKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]{2,4}$`)
	// DefaultWorkspaceRoles are the roles which exist in all workspaces regardless
	DefaultWorkspaceRoles = map[string]string{
		WorkspaceOwnerRole:                "Manages users, groups, access policies and roles within the workspace",
		WorkspaceMemberRole:               "Member of the workspace. No workspace management permissions.",
		WorkspaceAdminRole:                "Super-user access to clusters in workspace",
		WorkspaceEditorRole:               "Super-user access to namespaces in Non-Production. View only access in Production",
		WorkspaceViewerRole:               "View only access to all clusters in workspace",
		WorkspaceTroubleshootingRole:      "Access to logs and exec into running containers",
		WorkspaceSecretsViewerProdRole:    "Access to secrets, and other sensitive K8s configs",
		WorkspaceSecretsViewerNonProdRole: "Access to secrets in Non-Production, and other sensitive K8s configs",
	}

	// DefaultWorkspaceOwnerRoles are the roles assigned to the person who creates the workspace
	DefaultWorkspaceOwnerRoles = []string{
		WorkspaceOwnerRole,
		WorkspaceMemberRole,
		WorkspaceAdminRole,
	}
)
