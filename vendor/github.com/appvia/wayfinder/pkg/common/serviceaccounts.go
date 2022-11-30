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

const (
	// ManagedClusterWFServiceAccount is the name of the service account created in wayfinder
	// built clusters
	// @NOTE if this is changed is has ramifications for policies - please check in the
	// pkg/wayfinder/assets/policy folder
	ManagedClusterWFServiceAccount = "wf-cluster-admin"

	// WayfinderServiceAccount is the name of the kubernetes service account
	// Note this has to be the same as the name defined in the wayfinder chart
	WayfinderServiceAccount = "wayfinder-admin"
)
