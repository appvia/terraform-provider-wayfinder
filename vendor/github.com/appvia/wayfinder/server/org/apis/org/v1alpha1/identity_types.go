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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// IdentitySpec defines the desired state of User
type IdentitySpec struct {
	// AccountType is the account type of the identity i.e. sso, basicauth etc
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	AccountType string `json:"accountType"`
	// BasicAuth defines a basicauth identity
	// +kubebuilder:validation:Optional
	BasicAuth *BasicAuth `json:"basicAuth,omitempty"`
	// IDPUser links to the associated idp user
	// +kubebuilder:validation:Optional
	IDPUser *IDPUser `json:"idpUser,omitempty"`
	// User is the user spec the identity is associated
	// +kubebuilder:validation:Required
	User *User `json:"user"`
}

// BasicAuth defines the basicauth identity
type BasicAuth struct {
	// Password is a password associated to the user
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Password string `json:"password"`
}

// IDPUser is associated idp user
type IDPUser struct {
	// Email for the associated user
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Email string `json:"email"`
	// UUID is a unique id for the user in the external idp
	// +kubebuilder:validation:Optional
	UUID string `json:"uuid,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Identity is the Schema for the identities API
// +kubebuilder:resource:path=identities
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
