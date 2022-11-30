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
	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// SystemNamespacePrefix is the prefix used for system namespaces where wayfinder deploys and
	// runs its own workloads
	SystemNamespacePrefix = "wf-"

	// SystemNamespace is the system namespace
	SystemNamespace = SystemNamespacePrefix + "system"
	// OperatorsNamespace is the namespace for operators
	OperatorsNamespace = SystemNamespacePrefix + "operators"

	// ManagedClusterWFNamespace is the namespace Wayfinder creates and uses for its own core resources
	// in clusters that it manages
	ManagedClusterWFNamespace = SystemNamespacePrefix + "manager"
	// ManagedClusterSystemNamespace is the system namespace in managed clusters
	ManagedClusterSystemNamespace = SystemNamespacePrefix + "system"
	// ManagedClusterOperatorsNamespace is the namespace for operators in managed clusters
	ManagedClusterOperatorsNamespace = SystemNamespacePrefix + "operators"
	// ManagedClusterBuildsNamespace is the namespace for kaniko builds
	ManagedClusterBuildsNamespace = SystemNamespacePrefix + "builds"
)

// Namespace is the namespace wayfinder itself is running in
var Namespace = "wayfinder"

// WorkspaceNamespace returns the namespace in the management cluster which is used to store resources
// for the workspace
func WorkspaceNamespace(ws corev1.WorkspaceKey) string {
	return ws.Namespace()
}
