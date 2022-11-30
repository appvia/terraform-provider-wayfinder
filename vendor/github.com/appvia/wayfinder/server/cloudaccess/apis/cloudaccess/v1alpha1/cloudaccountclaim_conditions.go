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

package v1alpha1

import (
	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// ConditionClaimParentValid will be set for all claims that have a valid parent account
	ConditionClaimParentValid corev1alpha1.ConditionType = "ParentValid"

	// ConditionClaimManagedAccountNameAssigned will be set when the name of a managed account is
	// retrieved. If the name does not exist a name will be created
	ConditionClaimManagedAccountNameAssigned corev1alpha1.ConditionType = "ManagedAccountNameAssigned"

	// ConditionClaimManagedAccountProvisioned will be set when a managed account has been requested
	// and provisioned
	ConditionClaimManagedAccountProvisioned corev1alpha1.ConditionType = "ManagedAccountProvisioned"

	// ConditionClaimAccountReady is set when it is asserted that the account is ready for use, this
	// will be set by both managed and unmanaged account claims
	ConditionClaimAccountReady corev1alpha1.ConditionType = "AccountReady"
)

var ManagedCloudAccountClaimConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionClaimManagedAccountNameAssigned, Name: "Account name assigned"},
	{Type: ConditionClaimManagedAccountProvisioned, Name: "Managed account provisioned"},
}

var CloudAccountClaimConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionClaimParentValid, Name: "Parent account/organization valid"},
	{Type: ConditionClaimAccountReady, Name: "Referenced cloud account ready"},
	{Type: corev1alpha1.ConditionReady, Name: "Cloud account claim ready"},
}
