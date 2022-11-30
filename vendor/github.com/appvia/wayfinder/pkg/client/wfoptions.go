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

package client

import corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"

type ListOptions struct {
	InWorkspace corev1.WorkspaceKey
}

type DeleteOptions struct {
	DryRun  bool
	Orphan  bool
	Cascade bool
	Force   bool
}

type CreateOptions struct {
	DryRun bool
}

type UpdateOptions struct {
	DryRun            bool
	NoRetryOnConflict bool
	Force             bool
}

type ListOption interface {
	// ApplyToList applies this configuration to the given options.
	ApplyToList(*ListOptions)
}

type DeleteOption interface {
	// ApplyToDelete applies this configuration to the given options.
	ApplyToDelete(*DeleteOptions)
}

type CreateOption interface {
	// ApplyToCreate applies this configuration to the given options.
	ApplyToCreate(*CreateOptions)
}

type UpdateOption interface {
	// ApplyToUpdate applies this configuration to the given options.
	ApplyToUpdate(*UpdateOptions)
}

func GetListOpts(opts []ListOption) ListOptions {
	lo := &ListOptions{}
	for _, o := range opts {
		o.ApplyToList(lo)
	}
	return *lo
}

func GetCreateOpts(opts []CreateOption) CreateOptions {
	co := &CreateOptions{}
	for _, o := range opts {
		o.ApplyToCreate(co)
	}
	return *co
}

func GetUpdateOpts(opts []UpdateOption) UpdateOptions {
	uo := &UpdateOptions{}
	for _, o := range opts {
		o.ApplyToUpdate(uo)
	}
	return *uo
}

func GetDeleteOptions(opts []DeleteOption) DeleteOptions {
	do := &DeleteOptions{}
	for _, o := range opts {
		o.ApplyToDelete(do)
	}
	return *do
}

type InWorkspace corev1.WorkspaceKey

func (n InWorkspace) ApplyToList(opts *ListOptions) {
	opts.InWorkspace = corev1.WorkspaceKey(n)
}

type WithOrphan bool

func (n WithOrphan) ApplyToDelete(opts *DeleteOptions) {
	opts.Orphan = bool(n)
}

type WithCascade bool

func (n WithCascade) ApplyToDelete(opts *DeleteOptions) {
	opts.Cascade = bool(n)
}

type WithDryRun bool

func (n WithDryRun) ApplyToDelete(opts *DeleteOptions) {
	opts.DryRun = bool(n)
}
func (n WithDryRun) ApplyToUpdate(opts *UpdateOptions) {
	opts.DryRun = bool(n)
}
func (n WithDryRun) ApplyToCreate(opts *CreateOptions) {
	opts.DryRun = bool(n)
}

type WithForce bool

func (n WithForce) ApplyToDelete(opts *DeleteOptions) {
	opts.Force = bool(n)
}
func (n WithForce) ApplyToUpdate(opts *UpdateOptions) {
	opts.Force = bool(n)
}

// WithNoRetryOnConflict will prevent automatically retrying an update where the server responds
// with an 'object modified' error. The standard behaviour is to update the ResourceVersion on
// conflict, provided the Generation of the object remains the same.
type WithNoRetryOnConflict bool

func (n WithNoRetryOnConflict) ApplyToUpdate(opts *UpdateOptions) {
	opts.NoRetryOnConflict = bool(n)
}
