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
)

// identity provides a generate implementation for user identity
// we need to everything over this
type identity struct {
	address      string
	disabled     bool
	email        string
	method       string
	scopes       []string
	globalGroups []string
	workspaces   map[corev1.WorkspaceKey][]string
	username     string

	// uid is only typically set for robot users
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

// WithEmail sets the email addresss
func WithEmail(v string) IdentityFunc {
	return func(i *identity) {
		i.email = v
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

func WithGlobalGroup(group ...string) IdentityFunc {
	return func(i *identity) {
		i.globalGroups = append(i.globalGroups, group...)
	}
}

// WithUID populates the UID on this identity, typically only used for robots
func WithUID(uid string) IdentityFunc {
	return func(i *identity) {
		i.uid = uid
	}
}

// New creates and validates a new identity
func New(options ...IdentityFunc) Identity {
	i, _ := NewIdentity(options...)
	return i
}

// NewIdentity creates and validates a new identity
func NewIdentity(options ...IdentityFunc) (Identity, error) {
	i := &identity{workspaces: make(map[corev1.WorkspaceKey][]string)}

	for _, x := range options {
		x(i)
	}

	validErr := i.IsValid()
	// Populate default groups / workspace groups
	if validErr == nil {
		populateImplicitGroups(i)
	}

	return i, validErr
}

func populateImplicitGroups(i *identity) {
	if i.IsRobot() {
		i.globalGroups = append(i.globalGroups, common.WayfinderImplicitRobotGroup)
		for ws := range i.workspaces {
			i.workspaces[ws] = append(i.workspaces[ws], common.WorkspaceImplicitRobotGroup)
		}
		return
	}
	if i.IsUser() {
		i.globalGroups = append(i.globalGroups, common.WayfinderImplicitUserGroup)
		if i.IsGlobalAdmin() {
			i.globalGroups = append(i.globalGroups, common.WayfinderImplicitAdminGroup)
		}
		for ws := range i.workspaces {
			i.workspaces[ws] = append(i.workspaces[ws], common.WorkspaceImplicitUserGroup)
		}
	}
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
	return i.InGlobalGroup(common.WayfinderImplicitAdminGroup)
}

func (i *identity) IsRobot() bool {
	return utils.Contains(ScopeRobot, i.scopes)
}

func (i *identity) IsUser() bool {
	return utils.Contains(ScopeUser, i.scopes)
}

// IsMember checks if the identity is a member of a workspace
func (i *identity) IsMember(workspace corev1.WorkspaceKey) bool {
	return i.Workspaces().Contains(workspace)
}

// Email returns the identity email
func (i *identity) Email() string {
	return i.email
}

// Disabled checks if the identity is disabled
func (i *identity) Disabled() bool {
	return i.disabled
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

// WorkspaceGroups is a collection of groups the identity has in the specified workspace
func (i *identity) WorkspaceGroups(workspace corev1.WorkspaceKey) []string {
	return i.workspaces[workspace]
}

// GlobalGroups is the collection of groups the identity has to Wayfinder as a whole
func (i *identity) GlobalGroups() []string {
	return i.globalGroups
}

// InGlobalGroup will tell you if the user is a member of the specified global group
func (i *identity) InGlobalGroup(group string) bool {
	return utils.Contains(group, i.GlobalGroups())
}

// InWorkspaceGroup tells you if the specified user is a member of specified group in the workspace
// specified workspace.
func (i *identity) InWorkspaceGroup(workspace corev1.WorkspaceKey, group string) bool {
	return utils.Contains(group, i.WorkspaceGroups(workspace))
}

// UID returns the associated UID for this identity - only typically used with robots
func (i *identity) UID() string {
	return i.uid
}
