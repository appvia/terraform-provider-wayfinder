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
	"fmt"
	"strings"

	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/appvia/wayfinder/pkg/authentication"
	"github.com/appvia/wayfinder/pkg/utils"
	policyv1alpha1 "github.com/appvia/wayfinder/server/policydelivery/apis/policy/v1alpha1"
	policydeliveryv1alpha1 "github.com/appvia/wayfinder/server/policydelivery/apis/policydelivery/v1alpha1"
)

const robotExchangeSuffix = ":exchange"

// GenerateRobotAccount generates a robot account name
func GenerateRobotAccount(namespace, name string) string {
	return fmt.Sprintf("%s:%s:%s", authentication.RobotAccountPrefix(), namespace, name)
}

// GenerateRobotAccountExchange generates a robot account name
func GenerateRobotAccountExchange(namespace, name string) string {
	return GenerateRobotAccount(namespace, name) + robotExchangeSuffix
}

// TrimExchangeFromRobotAccount returns the exchange
func TrimExchangeFromRobotAccount(name string) string {
	return strings.TrimSuffix(name, robotExchangeSuffix)
}

// GetRobotAccountName returns the robot name
func GetRobotAccountName(name string) string {
	return utils.GetItemIfExists(strings.Split(name, ":"), 4)
}

// GetRobotAccountNamespace returns the robot namespace
func GetRobotAccountNamespace(name string) string {
	return utils.GetItemIfExists(strings.Split(name, ":"), 3)
}

// IsRobotAccount checks if the subject is a robot account
func IsRobotAccount(subject string) bool {
	return strings.HasPrefix(subject, authentication.RobotAccountPrefix()+":")
}

// IsSubjectRobotAccount checks if the kind and the name are equivalent to a robot account
func IsSubjectRobotAccount(s policydeliveryv1alpha1.Subject) bool {
	return s.Kind == policydeliveryv1alpha1.SubjectKindRobot
}

// IsRobotEqual checks that the subject is equivalent to the specified Robot object
func IsRobotEqual(r *policyv1alpha1.Robot, s policydeliveryv1alpha1.Subject) bool {
	isRobot, robotRef := GetRobotObject(s)
	if !isRobot {
		return false
	}
	return r.Name == robotRef.Name && r.Namespace == robotRef.Namespace
}

// GetRobotObject will return an empty Robot object with the correct namespace and name
// The robot name is parsed from the subject name
// will return false and nil if the subject is not a robot or cannot be parsed
func GetRobotObject(s policydeliveryv1alpha1.Subject) (bool, *policyv1alpha1.Robot) {
	if !IsSubjectRobotAccount(s) {
		return false, nil
	}
	return true, &policyv1alpha1.Robot{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Workspace.Namespace(),
		},
	}
}
