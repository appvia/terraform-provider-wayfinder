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
	"errors"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
	"github.com/appvia/wayfinder/pkg/common"
	"github.com/appvia/wayfinder/pkg/utils"
	timeutils "github.com/appvia/wayfinder/pkg/utils/time"
	policyv1alpha1 "github.com/appvia/wayfinder/server/policydelivery/apis/policy/v1alpha1"
)

// MergeAdmissonRequests is used to merged a collection of request into a policy
func MergeAdmissonRequests(requests *policyv1alpha1.AdmissionRequestList, current *policyv1alpha1.Policy, workspace corev1.WorkspaceKey) error {

	findGroupSelector := func(group string, selectors []*policyv1alpha1.Selector) *corev1.ResourceSelector {

		for i := 0; i < len(selectors); i++ {
			if selectors[i].Resource == nil {
				continue
			}

			if utils.Contains(group, selectors[i].Resource.Groups) {
				return selectors[i].Resource
			}
		}

		return nil
	}

	findNamespaceSelector := func(selectors []*policyv1alpha1.Selector) *metav1.LabelSelector {

		for i := 0; i < len(selectors); i++ {
			if selectors[i].Namespace != nil {
				return selectors[i].Namespace
			}
		}

		return nil
	}

	var namespaces []string

	// @logic: the approach will be simple for now, we separate the apigroups
	// into individual sections and allows all verbs on the resources which have
	// be used
	for i := 0; i < len(requests.Items); i++ {
		if requests.Items[i].Kind.Group == "" {
			continue
		}

		namespaces = append(namespaces, requests.Items[i].Namespace)
		resources := []string{requests.Items[i].Kind.Resource}

		// @step: fix up for specific resource types
		switch {
		case utils.Contains(requests.Items[i].Kind.Group, []string{"apps", "extensions"}):
			if requests.Items[i].Kind.Resource == "deployments" {
				resources = append(resources, []string{"deployments/scale", "deployments/rollback"}...)
			}
		}

		rg := findGroupSelector(requests.Items[i].Kind.Group, current.Spec.Selectors)
		switch rg {
		case nil:
			current.Spec.Selectors = append(current.Spec.Selectors, &policyv1alpha1.Selector{
				Resource: &corev1.ResourceSelector{
					Groups:    []string{requests.Items[i].Kind.Group},
					Resources: resources,
					Verbs:     []string{"*"},
				},
			})

		default:
			if !utils.Contains(requests.Items[i].Kind.Resource, rg.Resources) {
				rg.Resources = utils.Unique(append(rg.Resources, resources...))
				sort.Strings(rg.Resources)
			}
		}
	}
	namespaces = utils.Unique(namespaces)
	sort.Strings(namespaces)

	ns := findNamespaceSelector(current.Spec.Selectors)
	if ns == nil {
		current.Spec.Selectors = append(current.Spec.Selectors, &policyv1alpha1.Selector{
			Namespace: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      common.LabelWorkspace,
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{workspace.Key()},
					},
					{
						Key:      common.LegacyLabelName,
						Operator: metav1.LabelSelectorOpIn,
						Values:   namespaces,
					},
				},
			},
		})
	} else {
		list := utils.Unique(append(ns.MatchExpressions[1].Values, namespaces...))
		sort.Strings(list)
		ns.MatchExpressions[1].Values = list
	}

	return nil
}

// IsCategory checks if the resource is a category
func IsCategory(annotations map[string]string, category string) bool {
	return annotations[policyv1alpha1.CategoryAnnotation] == category
}

// IsEnabled checks if the resource is enabled
func IsEnabled(annotations map[string]string) bool {
	return annotations[common.AnnotationEnabled] == "true" || annotations[common.AnnotationEnabled] == ""
}

// IsLearning checks if the resource is system owned
func IsLearning(annotations map[string]string) bool {
	return annotations[policyv1alpha1.CategoryAnnotation] == "learning"
}

// IsSystemOwned checks if the resource is system owned
func IsSystemOwned(annotations map[string]string) bool {
	return annotations[common.AnnotationSystem] == "true"
}

// IsReadOnly checks if the resource is readonly
func IsReadOnly(annotations map[string]string) bool {
	return annotations[common.AnnotationReadOnly] == "true"
}

// IsReadOnlyPolicy checks if the policy is readonly
func IsReadOnlyPolicy(policy metav1.Object) bool {
	return policy.GetAnnotations()[common.AnnotationReadOnly] == "true"
}

// IsSystemPolicy checks if the policy is a system policy
func IsSystemPolicy(policy metav1.Object) bool {
	return policy.GetAnnotations()[common.AnnotationSystem] == "true"
}

// IsCompliance checks if compliance names match
func IsCompliance(policy metav1.Object, name string) bool {
	return policy.GetAnnotations()[policyv1alpha1.ComplianceAnnotation] == name
}

// IsPolicyEnabled checks if the policy is enabled - if the annotation is not
// present we default to true
func IsPolicyEnabled(policy *policyv1alpha1.Policy) bool {
	if policy == nil {
		return false
	}

	// @step: check if the policy is being deleted
	if !policy.DeletionTimestamp.IsZero() {
		return false
	}

	annontatons := policy.GetAnnotations()
	if annontatons == nil {
		return true
	}

	// @step: check if the policy has a ttl and if the policy is up for eviction
	expiration, found := annontatons[policyv1alpha1.TimeToLiveAnnotation]
	if found {
		if tm, valid := timeutils.ParseUnixTimestamp(expiration); valid {
			if time.Now().After(tm) {
				return false
			}
		}
	}

	// @step: check if the annotation is false
	switch annontatons[common.AnnotationEnabled] {
	case "", "true":
		return true
	}

	// @step: if the policy is based on a plan we don't need to include
	// this one
	if policy.Spec.PolicyRef != nil {
		return false
	}

	return false
}

// IsPolicyPlanEnabled checks the plan is enabled
func IsPolicyPlanEnabled(plan *policyv1alpha1.PolicyPlan) bool {

	if plan == nil {
		return false
	}
	annotations := plan.GetAnnotations()
	if annotations == nil {
		return true
	}

	switch annotations[common.AnnotationEnabled] {
	case "", "true":
		return true
	}

	return false
}

// HasTarget check if the policy is hitting the target
func HasTarget(policy *policyv1alpha1.Policy, labels []string) bool {
	return policy.Spec.Target == nil
}

// HasSubjects checks if the subject exists
func HasSubjects(policy *policyv1alpha1.Policy, subjects []string) bool {

	if utils.Contains("*", subjects) {
		return true
	}

	list := append([]string{"*"}, subjects...)
	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Subject != nil {
			if utils.ContainsMatchingItems(list, policy.Spec.Selectors[i].Subject.Subjects) {
				return true
			}
		}
	}

	return false
}

// IsUsePolicy checks if the policy is providing a use verb
func IsUsePolicy(policy *policyv1alpha1.Policy) bool {

	list := []string{"*", policyv1alpha1.UseVerb}

	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Resource != nil {
			if utils.ContainsMatchingItems(list, policy.Spec.Selectors[i].Resource.Verbs) {
				return true
			}
		}
	}

	return false
}

// IsPolicyOwned check if the owner references is another policy
func IsPolicyOwned(policy *policyv1alpha1.Policy) bool {

	if len(policy.OwnerReferences) <= 0 {
		return false
	}

	apiGroup := policyv1alpha1.SchemeGroupVersion.String()

	for i := 0; i < len(policy.OwnerReferences); i++ {
		if policy.OwnerReferences[i].APIVersion == apiGroup && policy.OwnerReferences[i].Kind == "Policy" {
			return true
		}
	}

	return false
}

// IsPolicyOwner checks if the policy is owned by another policy x
func IsPolicyOwner(policy *policyv1alpha1.Policy, owner string) bool {

	for _, x := range policy.OwnerReferences {
		if x.APIVersion != policyv1alpha1.SchemeGroupVersion.String() || x.Kind != "Policy" {
			continue
		}
		if x.Name == owner {
			return true
		}
	}

	return false
}

// IsAssumingPolicy checks if the policy is a generate role assumption
func IsAssumingPolicy(policy *policyv1alpha1.Policy) bool {
	annotations := policy.GetAnnotations()
	if annotations == nil {
		return false
	}

	return annotations[policyv1alpha1.GeneratedAnnotation] == policyv1alpha1.GeneratedByAssume
}

// IsAssignmentPolicy checks if the policy is an assignment policy
func IsAssignmentPolicy(policy *policyv1alpha1.Policy) bool {
	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Resource != nil {
			resource := policy.Spec.Selectors[i].Resource

			switch {
			case !utils.ContainsMatchingItems([]string{"*", policyv1alpha1.UseVerb}, resource.Verbs):
				continue
			case !utils.ContainsMatchingItems([]string{"*", "policyplans/assign", "policy/assign"}, resource.Resources):
				continue
			case !utils.ContainsMatchingItems([]string{"*", policyv1alpha1.GroupName}, resource.Groups):
				continue
			}

			return true
		}
	}

	return false
}

// IsAssumePolicy checks if the policy is providing an assume capability
func IsAssumePolicy(policy *policyv1alpha1.Policy) bool {
	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Resource != nil {
			resource := policy.Spec.Selectors[i].Resource

			switch {
			case !utils.ContainsMatchingItems([]string{"*", policyv1alpha1.UseVerb}, resource.Verbs):
				continue
			case !utils.ContainsMatchingItems([]string{"*", "policyplans/assume", "policy/assume"}, resource.Resources):
				continue
			case !utils.ContainsMatchingItems([]string{"*", policyv1alpha1.GroupName}, resource.Groups):
				continue
			}

			return true
		}
	}

	return false
}

// IsComplianceName checks if policy comes for a compliance package
func IsComplianceName(policy *policyv1alpha1.Policy, names []string) bool {

	annotations := policy.GetAnnotations()
	if annotations == nil {
		return false
	}

	return utils.Contains(annotations[policyv1alpha1.ComplianceAnnotation], names)

}

// IsRolePolicyPlan checks if policy is a role plan
func IsRolePolicyPlan(policy *policyv1alpha1.PolicyPlan) bool {
	annotations := policy.GetAnnotations()
	if annotations == nil {
		return false
	}

	return annotations[policyv1alpha1.CategoryAnnotation] == "role"
}

// HasPlans checks if the plans exists
func HasPlans(policy *policyv1alpha1.Policy, plans []string) bool {

	if policy.Spec.PolicyRef == nil {
		return false
	}

	return utils.Contains(policy.Spec.PolicyRef.Name, plans)
}

// HasRoles checks the roles
func HasRoles(policy *policyv1alpha1.Policy, roles []string) bool {

	if utils.Contains("*", roles) {
		return true
	}

	list := append([]string{"*"}, roles...)
	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Subject != nil {
			if utils.ContainsMatchingItems(list, policy.Spec.Selectors[i].Subject.Roles) {
				return true
			}
		}
	}

	return false
}

// HasGroups checks if the groups exists
func HasGroups(policy *policyv1alpha1.Policy, groups []string) bool {

	list := append([]string{"*"}, groups...)

	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Subject != nil {
			if utils.ContainsMatchingItems(list, policy.Spec.Selectors[i].Subject.Groups) {
				return true
			}
		}
	}

	return false
}

// HasScopes checks if the scopes exists
func HasScopes(policy *policyv1alpha1.Policy, scopes []string) bool {

	if utils.Contains("*", scopes) {
		return true
	}

	list := append([]string{"*"}, scopes...)
	for i := 0; i < len(policy.Spec.Selectors); i++ {
		if policy.Spec.Selectors[i].Subject != nil {
			if utils.ContainsMatchingItems(list, policy.Spec.Selectors[i].Subject.Scopes) {
				return true
			}
		}
	}

	return false
}

// GetAssignmentValue returns the assignment value
func GetAssignmentValue(name string, assignment *policyv1alpha1.Assignment) (string, bool, error) {

	if assignment == nil {
		return "", false, nil
	}

	for i := 0; i < len(assignment.Spec.Inputs); i++ {
		if assignment.Spec.Inputs[i].Name == name {
			switch {
			case assignment.Spec.Inputs[i].Value != nil:
				return utils.StringValue(assignment.Spec.Inputs[i].Value), true, nil

			case assignment.Spec.Inputs[i].Value == nil:
				return "", false, errors.New("has no value defined")

			case assignment.Spec.Inputs[i].Values != nil:
				return "", false, errors.New("has multiple values")
			}
		}
	}

	return "", false, nil
}
