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

import corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"

const (
	// ConditionUpdateApproved indicates if the update is blocked for confirmation
	// A user must confirm the update is safe to proceed
	ConditionUpdateApproved corev1alpha1.ConditionType = "UpdateApproved"
	// ConditionUpdatePreRequisitesComplete indicates if the update is blocked by prerequisite(s)
	// Note: there may be more prerequisites than one
	//       only one prerequisite is directly blocking this update
	// When the preceding prerequisite status is ready, the update will proceed
	ConditionUpdatePreRequisitesComplete corev1alpha1.ConditionType = "UpdatePreRequisitesComplete"
)

var UpdatePreConditions = []corev1alpha1.ConditionSpec{
	{Type: ConditionUpdatePreRequisitesComplete, Name: "Update prerequisite are met"},
	{Type: ConditionUpdateApproved, Name: "Update is approved"},
}
