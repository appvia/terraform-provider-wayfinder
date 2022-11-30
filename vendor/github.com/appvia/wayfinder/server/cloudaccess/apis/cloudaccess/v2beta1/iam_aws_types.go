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
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type AWSPolicy struct {
	Description    string        `json:"description"`
	Path           string        `json:"path"`
	PolicyName     string        `json:"policyName"`
	PolicyDocument apiextv1.JSON `json:"policyDocument"`
}

// SetPolicy to set the policy document as raw json
func (p *AWSPolicy) SetPolicyDocument(s string) error {
	if s == "" {
		p.PolicyDocument.Raw = nil

		return nil
	}
	p.PolicyDocument = apiextv1.JSON{Raw: []byte(s)}

	return nil
}

type AWSRole struct {
	AssumeRolePolicyDocument apiextv1.JSON `json:"assumeRolePolicyDocument"`
	Description              string        `json:"description"`
	Path                     string        `json:"path"`
	RoleName                 string        `json:"roleName"`
}

// SetPolicy to set the policy document as raw json
func (r *AWSRole) SetAssumePolicyDocument(s string) error {
	if s == "" {
		r.AssumeRolePolicyDocument.Raw = nil

		return nil
	}
	r.AssumeRolePolicyDocument = apiextv1.JSON{Raw: []byte(s)}

	return nil
}

type AWSRolePolicyAttachments struct {
	RoleName  string `json:"roleName"`
	PolicyArn string `json:"policyArn"`
}

// AWSRoleWithPolicies is a collection of AWS roles and policies
type AWSRoleWithPolicies struct {
	WayfinderRoleName     string                     `json:"wayfinderRoleName"`
	Role                  AWSRole                    `json:"role"`
	Policies              []AWSPolicy                `json:"policies"`
	RolePolicyAttachments []AWSRolePolicyAttachments `json:"rolePolicyAttachments"`
}

// AWSIAM is a collection of AWS roles and policies
type AWSIAM struct {
	AccountID string                `json:"accountID"`
	Features  []string              `json:"features"`
	IAM       []AWSRoleWithPolicies `json:"iam"`
}
