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
	// ConditionNodePoolUpdateDependenciesReady indicates the node pool is ready to be updated
	ConditionNodePoolUpdateDependenciesReady corev1alpha1.ConditionType = "ConditionNodePoolUpdateDependenciesReady"
	// ConditionNodePoolUpdateInProgress indicates the node pool is being updated
	ConditionNodePoolUpdateInProgress corev1alpha1.ConditionType = "NodePoolUpdateInProgress"
	// ConditionNodePoolUpdateAutoscalingDisabled indicates that autoscaling has been disabled
	ConditionNodePoolUpdateAutoscalingDisabled corev1alpha1.ConditionType = "NodePoolUpdateAutoscalingDisabled"
	// ConditionNodePoolUpdateMigrated indicates if the workloads have been migrated to the nodepool replacement
	ConditionNodePoolUpdateMigrated corev1alpha1.ConditionType = "NodePoolUpdateMigrated"
	// ConditionNodePoolUpdateOldNodePoolDeleted indicates if the nodepool replacement has been deleted
	ConditionNodePoolUpdateOldNodePoolDeleted corev1alpha1.ConditionType = "NodePoolUpdateOldNodePoolDeleted"
	// ConditionNodePoolUpdateAutoscalingEnabled indicates that autoscaling has been enabled
	ConditionNodePoolUpdateAutoscalingEnabled corev1alpha1.ConditionType = "NodePoolUpdateAutoscalingEnabled"
)
