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

package v2beta1

import (
	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// ConditionCloudIdentityPrepared will be set for workload identities once the cloud-level
	// identity is prepared and ready
	ConditionCloudIdentityPrepared corev1alpha1.ConditionType = "CloudIdentityPrepared"
)

var WorkloadIdentityConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionCloudIdentityPrepared, Name: "Cloud identity prepared"},
	{Type: corev1alpha1.ConditionReady, Name: "Workload Identity ready"},
}
