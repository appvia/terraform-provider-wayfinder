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
	// RegistryProviderECR represents Amazon Web Services' Elastic Container Registry
	RegistryProviderECR = "ecr"
	// RegistryProviderHarbor represents go-harbor
	RegistryProviderHarbor = "harbor"
)

// RegistryProviders is the set of supported registry providers in Wayfinder
var RegistryProviders = []string{
	RegistryProviderECR,
	RegistryProviderHarbor,
}
