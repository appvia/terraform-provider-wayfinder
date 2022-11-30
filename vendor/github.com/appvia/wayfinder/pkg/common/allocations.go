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
	"context"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// GetAllocationPrefixes returns a map of common allocation prefixes, which
// will hopefully be deprecated once we relook at allocations as a whole
func GetAllocationPrefixes() map[string]string {
	return map[string]string{
		"AKS": "cloudaccount-azure",
		"EKS": "cloudaccount-aws",
		"GKE": "cloudaccount-gcp",
	}
}

// AllocationChecker represents a function which checks that a workspace is allowed to use thing
type AllocationChecker func(ctx context.Context, workspace string, thing corev1alpha1.Ownership) (bool, error)
