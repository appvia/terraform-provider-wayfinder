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
	// PodSecurityStandardPrivileged is the name of the pod security standard policy Privileged
	PodSecurityStandardPrivileged = "privileged"
	// PodSecurityStandardBaseline is the name of the pod security standard policy Baseline
	PodSecurityStandardBaseline = "baseline"
	// PodSecurityStandardRestricted is the name of the pod security standard policy Restricted
	PodSecurityStandardRestricted = "restricted"
)

const (
	PodSecurityStandardKubernetesIO = "pod-security.kubernetes.io"
)

const (
	// LabelPodSecurityStandardEnforce is the value used to label a namespace with enforce policy name
	LabelPodSecurityStandardEnforce = PodSecurityStandardKubernetesIO + "/enforce"
	// LabelPodSecurityStandardEnforceVersion is the value used to label a namespace with enforce policy version
	LabelPodSecurityStandardEnforceVersion = PodSecurityStandardKubernetesIO + "/enforce-version"
	// LabelPodSecurityStandardWarn is the value used to label a namespace with warn policy name
	LabelPodSecurityStandardWarn = PodSecurityStandardKubernetesIO + "/warn"
	// LabelPodSecurityStandardWarnVersion is the value used to label a namespace with warn policy version
	LabelPodSecurityStandardWarnVersion = PodSecurityStandardKubernetesIO + "/warn-version"
	// LabelPodSecurityStandardAudit is the value used to label a namespace with audit policy name
	LabelPodSecurityStandardAudit = PodSecurityStandardKubernetesIO + "/audit"
	// LabelPodSecurityStandardAuditVersion is the value used to label a namespace with audit policy version
	LabelPodSecurityStandardAuditVersion = PodSecurityStandardKubernetesIO + "/audit-version"
)

const (
	// PodSecurityStandardVersionLatest is the name of the pod security standard policy version latest
	PodSecurityStandardVersionLatest = "latest"
)
