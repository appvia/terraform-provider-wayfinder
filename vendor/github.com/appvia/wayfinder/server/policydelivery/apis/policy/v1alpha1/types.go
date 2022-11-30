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
	"fmt"
	"net/http"
	"net/url"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// GeneratedByAssume is an annotation added on assume policies - used for filtering
	GeneratedByAssume = "assume"
	// GenerateByAssignment is an annotation added on generated assignments - used for filtering
	GenerateByAssignment = "assignment"
	// GenerateByDeletion is an annotation added to generated deletion policies - use for filtering
	GenerateByDeletion = "deletion"
)

const (
	// BoundaryCategory is associated to any policy around scoping robot learning
	BoundaryCategory = "boundary"
	// ConstraintCategory is associated to any policy associated to security policy
	ConstraintCategory = "constraint"
	// DeploymentCategory is associated to any policy associated to deploying into a cluster
	DeploymentCategory = "deployment"
	// WayfinderCategory is associated to any policy used to speak to the Wayfinder API
	WayfinderCategory = "wayfinder"
	// SystemCategory is associated to any policy which is internal
	SystemCategory = "system"
)

const (
	// CategoryAnnotation is an annotation used to define the category
	CategoryAnnotation = "policies.appvia.io/category"
	// ComplianceAnnotation is the bundle the policy comes from
	ComplianceAnnotation = "policies.appvia.io/compliance"
	// GenerateAnnotation indicates the policy was generate by (assumed, assignment)
	GeneratedAnnotation = "policies.appvia.io/generated"
	// InternalAnnotation indicates the plan can never be assigned
	InternalAnnotation = "policies.appvia.io/internal"
	// MustMatchAnnotation indicates the policy must match
	MustMatchAnnotation = "policies.appvia.io/mustmatch"
	// PermissiveAnnotation indicates the policy is permissive
	PermissiveAnnotation = "policies.appvia.io/permissive"
	// PolicyNameAnnotation is the policy name
	PolicyNameAnnotation = "policies.appvia.io/policy"
	// PlanNameAnnotation is the name of the plan the policy was derived
	PlanNameAnnotation = "policies.appvia.io/plan"
	// PlanRevisionAnnotation is the revision of the plan which was used
	PlanRevisionAnnotation = "policies.appvia.io/revision"
	// RoleAnnotation indicates the policy can been used a role
	RoleAnnotation = "policies.appvia.io/role"
	// RobotAnnotation indicates the robot this policy is associated to
	RobotAnnotation = "policies.appvia.io/robot"
	// ScopeAnnotation indicates the intended scope
	ScopeAnnotation = "policies.appvia.io/scope"
	// TimeToLiveAnnotation indicates the policy has a time-to-live
	TimeToLiveAnnotation = "policies.appvia.io/ttl"
)

const (
	// NamespaceScope indicates this should be applied to all workspaces
	NamespaceScope = "namespace"
)

const (
	// PolicyLabel is added to generate policies
	PolicyLabel = "policy.policies.appvia.io"
	// PlanLabel is added to generate policies
	PlanLabel = "plan.policies.appvia.io"
	// PolicyClusterLabel is cluster label for a policy
	PolicyClusterLabel = "cluster.policies.appvia.io"
	// PolicyTypeLabel is added to generate policies
	PolicyTypeLabel = "policy_type.policies.appvia.io"
	// PolicyRobotLabel is robot account uid
	PolicyRobotLabel = "robot.policies.appvia.io"
)

const (
	// PolicyAssigmentTypeLabel indicates an assignment
	PolicyAssigmentTypeLabel = "assignment"
	// PolicyProfiledTypeLabel indicates a profile label
	PolicyProfiledTypeLabel = "profiled"
)

const (
	// BindVerb allows for policy association
	BindVerb = "bind"
	// CreateVerb indicates a new resource
	CreateVerb = "create"
	// DeleteVerb removes a resource
	DeleteVerb = "delete"
	// EscalateVerb indicates we can use a resource
	EscalateVerb = "escalate"
	// GetVerb is the retrieval
	GetVerb = "get"
	// LearnVerb is used internally for profiling
	LearnVerb = "learn"
	// ListVerb returns a listing
	ListVerb = "list"
	// PatchVerb indicates we can use a resource
	PatchVerb = "patch"
	// UpdateVerb updates a resource
	UpdateVerb = "update"
	// UseVerb indicates we can use a resource
	UseVerb = "use"
)

// Verbs is a collection of rbac verbs
var Verbs = []string{
	BindVerb,
	CreateVerb,
	DeleteVerb,
	EscalateVerb,
	GetVerb,
	ListVerb,
	PatchVerb,
	UpdateVerb,
	UseVerb,
}

const (
	// AllowedDecision indicates the operation is allowed
	AllowedDecision = "allow"
	// DeniedDecision indicates the operation was denied
	DeniedDecision = "denied"
	// LearnDecision indicates the operation should be learnt
	LearnDecision = "learn"
	// LoggedDecision indicates the operation was allowed but logged with an error
	LoggedDecision = "logged"
)

// Target is where the policy should be applied, the apiserver, or remote
// cluster/s
type Target struct {
	// Selector defines the location of a policy - which can be placed on a plan,
	// workspace, cluster etc - effectively these all get plached into clusters
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// Selector provides a generate selector on resources
type Selector struct {
	// Namespace is a namespace selector
	// +kubebuilder:validation:Optional
	Namespace *metav1.LabelSelector `json:"namespace,omitempty"`
	// Resource selects on a kubernetes resource
	// +kubebuilder:validation:Optional
	Resource *corev1alpha1.ResourceSelector `json:"resource,omitempty"`
	// Subject is a subject selector
	// +kubebuilder:validation:Optional
	Subject *corev1alpha1.SubjectSelector `json:"subject,omitempty"`
}

// PlanPolicyRef is defines a reference to the policy plan that was used
// to create this policy
type PlanPolicyRef struct {
	// Name is the name of the policy plan
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Version is a hash of the policy plan configuration so we know
	// when we have strays from the version
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Version string `json:"version"`
}

// Decision is a inline decision on the outcome of the policy
type Decision struct {
	// Action is the decision outcome i.e. allowed, denied or logged
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Action string `json:"action"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// Message is a human readable reason for the outcome
	Message string `json:"message"`
}

// LogResult indicates the response should be logged
type LogResult struct {
	// Severity is the level of the event
	Severity string `json:"severity,omitempty"`
	// Message is the message which should be logged
	Message string `json:"message,omitempty"`
}

// DeniedResult indicates a denial error
// +kubebuilder:object:generate=false
type DeniedResult struct {
	// Policy is the name of the policy
	// +kubebuilder:validation:Optional
	Policy string `json:"policy,omitempty"`
	// Code is a machine readable code indicates the error
	// +kubebuilder:validation:Optional
	Code int `json:"code,omitempty"`
	// Field is the optional field in question
	// +kubebuilder:validation:Optional
	Field string `json:"field,omitempty"`
	// Message is a human readable message
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Message string `json:"message"`
	// Msg is a human readable message - added to make us compatible with gatekeeper
	Msg string `json:"msg,omitempty"`
	// Value is the current value of the field
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// ValidationErrorResult indicates a validation error was found
type ValidationErrorResult struct {
	// Policy is the name of the policy
	// +kubebuilder:validation:Optional
	Policy string `json:"policy,omitempty"`
	// Field is the optional field in question
	// +kubebuilder:validation:Optional
	Field string `json:"field,omitempty"`
	// Value is the current value of the field
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
	// Allowed is an optional permitted list
	// +kubebuilder:validation:Optional
	Allowed []string `json:"allowed,omitempty"`
	// Message is a human readable message
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Message string `json:"message"`
}

// String returns a string representation
func (v *ValidationErrorResult) String() string {
	message := v.Message

	if message == "" {
		message = "valiation error"
	}

	switch {
	case v.Field != "" && v.Value != "":
		return fmt.Sprintf("%s=%v %s", v.Field, v.Value, message)

	case v.Field != "":
		return fmt.Sprintf("%s is empty %s", v.Field, message)

	case v.Value != "":
		return fmt.Sprintf("%v %s", v.Value, message)
	}

	return message
}

// RolesResult indicates the policy as has permitted the use of a role based
// on the policy - this is largely used for rbac purposes
// +kubebuilder:object:generate=true
type RolesResult struct {
	// Policy is the name of the policy
	// +kubebuilder:validation:Optional
	Policy string `json:"policy,omitempty"`
	// Code is a machine readable code indicates the error
	// +kubebuilder:validation:Optional
	Code int `json:"code,omitempty"`
	// Roles is a collection of rbac roles which have been granted from the policy
	// +kubebuilder:validation:Required
	Roles []string `json:"roles"`
}

// AllowedResult indicates the policy we activately permitted by a policy
// +kubebuilder:object:generate=false
type AllowedResult struct {
	// Policy is the name of the policy
	// +kubebuilder:validation:Optional
	Policy string `json:"policy,omitempty"`
	// Code is a machine readable code indicates the error
	// +kubebuilder:validation:Optional
	Code int `json:"code,omitempty"`
	// Field is the optional field in question
	// +kubebuilder:validation:Optional
	Field string `json:"field,omitempty"`
	// Message is a human readable message
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Message string `json:"message"`
	// Value is the current value of the field
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// +k8s:deepcopy-gen=false
type Object map[string]interface{}

func (in *Object) DeepCopyInto(out *Object) {
	if in == nil {
		*out = nil
	} else {
		*out = runtime.DeepCopyJSON(*in)
	}
}

func (in *Object) DeepCopy() *Object {
	if in == nil {
		return nil
	}
	out := new(Object)
	in.DeepCopyInto(out)

	return out
}

// AdmissionRequest is request to evalute an access request
type AdmissionRequest struct {
	// DryRun indicates this is a dryrun to see the evaluation
	// +kubebuilder:validation:Optional
	DryRun *bool `json:"dryRun,omitempty"`
	// UUID is a unique id for the request
	// +kubebuilder:validation:Required
	UUID types.UID `json:"uuid"`
	// Kind is the fully-qualified resource being requested
	// +kubebuilder:validation:Required
	Kind metav1.GroupVersionResource `json:"kind"`
	// SubResource is the subresource being requested, if any (for example, "status" or "scale")
	// +kubebuilder:validation:Optional
	SubResource string `json:"subResource,omitempty"`
	// Verb is the action being request
	// +kubebuilder:validation:Optional
	Verb string `json:"verb"`
	// Resource is the actual request payload if any
	// +kubebuilder:validation:Optional
	Resource []byte `json:"resource,omitempty"`
	// Object is the decoded resource from above - this is required for the engine to be able
	// to target the fields
	Object Object `json:"object,omitempty"`
	// Name is the name of the resourc
	// +kubebuilder:validation:Optional
	Name string `json:"name"`
	// Namespace is the workspace's namespace in which the resource resides
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	// User is the details related to the user requesting the action
	// +kubebuilder:validation:Optional
	User *UserInfo `json:"user,omitempty"`
	// Origin is the origin of the request i.e ip address and so forth
	// +kubebuilder:validation:Optional
	Origin *RequestOrigin `json:"origin,omitempty"`
}

// Decision is the outcome of request which are broken down in a collection
// of categories - validation errors, violations (denials), logged indicated
// resource should be logged
type AdmissionResponse struct {
	// EnableProfiling indicates the access request should be profiled
	// +optional
	EnableProfiling *bool `json:"enableProfiling,omitempty"`
	// Allowed is a collection of policy whom gave a allowed gave an allowed decision
	// +optional
	Allowed []*AllowedResult `json:"allowed,omitempty"`
	// Role is a collection of roles which have been granted based on the policy
	// +optional
	Role []*RolesResult `json:"role,omitempty"`
	// Logging is a collection of logging requirements
	// +optional
	Logging []*LogResult `json:"logging,omitempty"`
	// Validation is a collection of validation errors
	// +optional
	Validation []*ValidationErrorResult `json:"validation,omitempty"`
	// Violation is a collection of violation access this resource
	// +optional
	Violation []*DeniedResult `json:"violation,omitempty"`
}

// UserInfo are details on the caller
type UserInfo struct {
	// The name that uniquely identifies this user among all active users.
	// +optional
	Username string `json:"username,omitempty"`
	// The names of groups this user is a part of.
	// +optional
	Groups []string `json:"groups,omitempty"`
	// Roles are the roles the user holds in the various workspaces
	// +optional
	Roles []string `json:"roles,omitempty"`
	// Scopes indicates the scope the token i.e. user, token etc
	// +optional
	Scopes []string `json:"scopes,omitempty"`
	// Any additional information provided by the authenticator.
	// +optional
	Extra map[string]ExtraValue `json:"extra,omitempty"`
	// Claims are jwt claims from the user token
	// +optional
	Claims apiextv1.JSON `json:"claims,omitempty"`
	// Attributes are additional attributes on the user
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

// RequestOrigin are details on the where the request came from
type RequestOrigin struct {
	// URL is the incoming request url
	// +required
	URL string `json:"url,omitempty"`
	// Headers are any optonal http headers from the request
	// +optional
	Headers http.Header `json:"headers,omitempty"`
	// Address is a external address of the request
	// +optional
	Address string `json:"address,omitempty"`
	// Query are query parameters to the request
	// +optional
	Query url.Values `json:"query,omitempty"`
}

// ExtraValue masks the value so protobuf can generate
type ExtraValue []string

func (t ExtraValue) String() string {
	return fmt.Sprintf("%v", []string(t))
}
