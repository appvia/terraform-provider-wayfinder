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

import "fmt"

const (
	// KubeResourceNamePrefix is the prefix for naming all wayfinder created kube resources
	KubeResourceNamePrefix = "wf"
	// CloudResourceNamePrefix represents generic cloud resource name prefix that is used or created by wayfinder
	CloudResourceNamePrefix = "wf"
)

// GetKubePrefixedName returns a kubernetes resource name string
func GetKubePrefixedName(original string) string {
	return fmt.Sprintf("%s-%s", KubeResourceNamePrefix, original)
}

// GetCloudPrefixedName returns a cloud resource name string
func GetCloudPrefixedName(original string) string {
	return fmt.Sprintf("%s-%s", CloudResourceNamePrefix, original)
}
