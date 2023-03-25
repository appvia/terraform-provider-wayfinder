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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// +kubebuilder:webhook:name=identities.org.appvia.io,mutating=false,path=/validate/org.appvia.io/identities,verbs=create;update;delete,groups="org.appvia.io",resources=identities,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=identities.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/identities,verbs=create;update,groups="org.appvia.io",resources=identities,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// IdentityKind is the identity crd name
const IdentityKind = "Identity"

// IdentityGVK is the GVK for identities
var IdentityGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    IdentityKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Identity is the Schema for the identities API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=identities,scope=Cluster,categories={wayfinder}
type Identity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IdentitySpec `json:"spec,omitempty"`
}

func (Identity) APIPath() string {
	return "identities"
}
func (c *Identity) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *Identity) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*Identity))
}
func (Identity) ListType() corev1alpha1.ObjectList {
	return &IdentityList{}
}

func (*Identity) GetCommonStatus() *corev1alpha1.CommonStatus {
	// No status on an identity, so nothing to return
	return nil
}

// IdentitySpec describes an identity profile for a subject
type IdentitySpec struct {
	// Subject identifies the Wayfinder user that this refers to. At this time, the only supported
	// subject kind is User.
	// +kubebuilder:validation:Required
	Subject Subject `json:"subject"`
	// AccountType is the account type of the identity
	// +kubebuilder:validation:Required
	AccountType IdentityAccountType `json:"accountType"`
	// Provider uniquely identifies the provider within the account type (e.g. auth0, azuread, etc)
	// +kubebuilder:validation:Optional
	Provider string `json:"provider,omitempty"`
	// ProviderUsername is the name of this user in the provider
	// +kubebuilder:validation:Required
	ProviderUsername string `json:"username"`
	// ProviderToken is a token to authenticate or identify this user with the provider, if required
	// +kubebuilder:validation:Optional
	ProviderToken string `json:"providerToken,omitempty"`
	// ProviderUID is the ID of the user in the provider, if required
	// +kubebuilder:validation:Optional
	ProviderUID string `json:"providerUID,omitempty"`
	// Email is the email of this user in the provider, if available
	// +kubebuilder:validation:Optional
	Email string `json:"email,omitempty"`
}

// +kubebuilder:validation:Enum=SSO;BasicAuth;Token
type IdentityAccountType string

const (
	IdentityAccountTypeSSO       IdentityAccountType = "SSO"
	IdentityAccountTypeBasicAuth IdentityAccountType = "BasicAuth"
	IdentityAccountTypeToken     IdentityAccountType = "Token"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IdentityList contains a list of User
type IdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Identity `json:"items"`
}

func (IdentityList) ObjectType() corev1alpha1.Object {
	return &Identity{}
}
func (l *IdentityList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *IdentityList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*IdentityList))
}

func (l *IdentityList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *IdentityList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Identity{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Identity)))
	}
}
