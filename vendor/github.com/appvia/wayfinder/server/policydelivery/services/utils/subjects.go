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

package utils

import (
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/appvia/wayfinder/server/policydelivery/apis/policydelivery/v1alpha1"
)

// GetKubeSubject translates our policy delivery subject into a Kube RBAC subject
func GetKubeSubject(subject v1alpha1.Subject) *rbacv1.Subject {
	rbacsubj := &rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: "rbac.authorization.k8s.io",
	}

	switch subject.Kind {
	case v1alpha1.SubjectKindRobot:
		rbacsubj.Name = GenerateRobotAccount(subject.Workspace.Namespace(), subject.Name)
		return rbacsubj
	case v1alpha1.SubjectKindUser, "":
		rbacsubj.Name = subject.Name
		return rbacsubj
	default:
		return nil
	}
}
