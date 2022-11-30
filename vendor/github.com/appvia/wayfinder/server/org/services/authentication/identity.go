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

package authentication

import (
	"errors"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils"
	orgv2 "github.com/appvia/wayfinder/server/org/apis/org/v2beta1"
)

// identity provides a generate implementation for user identity
// we need to everything over this
type identity struct {
	address         string
	disabled        bool
	method          string
	scopes          []string
	wayfinderGroups []string
	workspaces      map[corev1.WorkspaceKey][]string
	username        string
	subjectKind     orgv2.SubjectKind

	// uid is only typically set for access token users
	uid string
}

// IdentityFunc returns a identity func
type IdentityFunc func(*identity)

// WithAddress set the ip address of the user
func WithAddress(v string) IdentityFunc {
	return func(i *identity) {
		i.address = v
	}
}

// WithDisabled set the disable boolean
func WithDisabled(v bool) IdentityFunc {
	return func(i *identity) {
		i.disabled = v
	}
}

// WithAuthMethod sets the authentication method
func WithAuthMethod(v string) IdentityFunc {
	return func(i *identity) {
		i.method = v
	}
}

// IsDisabled checks if the identity is disabled
func IsDisabled(v bool) IdentityFunc {
	return func(i *identity) {
		i.disabled = v
	}
}

// Username is a unique username for this identity
func WithUsername(v string) IdentityFunc {
	return func(i *identity) {
		i.username = v
	}
}

func WithSubjectKind(v orgv2.SubjectKind) IdentityFunc {
	return func(i *identity) {
		i.subjectKind = v
	}
}

// WithScopes is a list of scope for the identity
func WithScopes(scopes []string) IdentityFunc {
	return func(i *identity) {
		i.scopes = scopes
	}
}

// WithWorkspace adds a workspaces and roles to the identity
func WithWorkspace(workspace corev1.WorkspaceKey, workspaceGroups []string) IdentityFunc {
	return func(i *identity) {
		i.workspaces[workspace] = workspaceGroups
	}
}

func WithWayfinderGroup(group ...string) IdentityFunc {
	return func(i *identity) {
		i.wayfinderGroups = append(i.wayfinderGroups, group...)
	}
}

// WithUID populates the UID on this identity, typically only used for access tokens
func WithUID(uid string) IdentityFunc {
	return func(i *identity) {
		i.uid = uid
	}
}

// NewIdentity creates and validates a new identity
func NewIdentity(options ...IdentityFunc) Identity {
	i := &identity{workspaces: make(map[corev1.WorkspaceKey][]string)}

	for _, x := range options {
		x(i)
	}

	return i
}

// IsValid checks the identity is valid
func (i *identity) IsValid() error {
	switch {
	case i.username == "":
		return errors.New("no username")

	case i.method == "":
		return errors.New("no authentication method")
	}

	return nil
}

// AuthMethod is the method the identity logged in with
func (i *identity) AuthMethod() string {
	return i.method
}

// Address returns the ip address
func (i *identity) Address() string {
	return i.address
}

// IsGlobalAdmin checks if the identity is a global admin
func (i *identity) IsGlobalAdmin() bool {
	return i.InWayfinderGroup(common.WayfinderAdminGroup)
}

func (i *identity) IsAccessToken() bool {
	return i.SubjectKind() == orgv2.SubjectKindPlaformAccessToken ||
		i.SubjectKind() == orgv2.SubjectKindWorkspaceAccessToken
}

func (i *identity) IsUser() bool {
	return i.SubjectKind() == orgv2.SubjectKindUser
}

// IsMember checks if the identity is a member of a workspace
func (i *identity) IsMember(workspace corev1.WorkspaceKey) bool {
	return i.Workspaces().Contains(workspace)
}

// IsWorkspaceOwner checks if the identity is an owner of a workspace
func (i *identity) IsWorkspaceOwner(workspace corev1.WorkspaceKey) bool {
	return utils.Contains(common.WorkspaceOwnerGroup, i.WorkspaceGroups(workspace))
}

// Disabled checks if the identity is disabled
func (i *identity) Disabled() bool {
	return i.disabled
}

// SubjectKind is the type of user represented by this identity
func (i *identity) SubjectKind() orgv2.SubjectKind {
	return i.subjectKind
}

// Username is a unique username for this identity
func (i *identity) Username() string {
	return i.username
}

// Scopes is a list of scope for the identity
func (i *identity) Scopes() []string {
	return i.scopes
}

// Workspaces is a list of workspaces for the user
func (i *identity) Workspaces() corev1.WorkspaceKeys {
	var list []corev1.WorkspaceKey

	for ws := range i.workspaces {
		list = append(list, ws)
	}

	return list
}

func (i *identity) AllWorkspaceGroups() map[corev1.WorkspaceKey][]string {
	return i.workspaces
}

// WorkspaceGroups is a collection of groups the identity has in the specified workspace
func (i *identity) WorkspaceGroups(workspace corev1.WorkspaceKey) []string {
	return i.workspaces[workspace]
}

// WayfinderGroups is the collection of groups the identity has to Wayfinder as a whole
func (i *identity) WayfinderGroups() []string {
	return i.wayfinderGroups
}

// InWayfinderGroup will tell you if the user is a member of the Wayfinder defined group of users
func (i *identity) InWayfinderGroup(group string) bool {
	return utils.Contains(group, i.WayfinderGroups())
}

// InWorkspaceGroup tells you if the specified user is a member of specified group in the workspace
// specified workspace.
func (i *identity) InWorkspaceGroup(workspace corev1.WorkspaceKey, group string) bool {
	return utils.Contains(group, i.WorkspaceGroups(workspace))
}

// UID returns the associated UID for this identity - only typically used with access tokens
func (i *identity) UID() string {
	return i.uid
}

func (i *identity) ToSubject() orgv2.Subject {
	return orgv2.Subject{Name: i.Username(), Kind: i.SubjectKind()}
}
