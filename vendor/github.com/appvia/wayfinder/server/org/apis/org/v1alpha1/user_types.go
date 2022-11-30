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

// UserSpec defines the desired state of User
// +k8s:openapi-gen=true
type UserSpec struct {
	// Disabled indicates if the user is disabled
	Disabled bool `json:"disabled"`
	// Email is the email for the user
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Email string `json:"email"`
	// Username is the userame or identity for this user
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Username string `json:"username"`
}

// UserStatus defines the observed state of User
// +k8s:openapi-gen=true
type UserStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User is the Schema for the users API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=users
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

func (User) APIPath() string {
	return "users"
}
func (u *User) Clone() corev1alpha1.Object {
	return u.DeepCopy()
}
func (u *User) CloneInto(o corev1alpha1.Object) {
	u.DeepCopyInto(o.(*User))
}
func (User) ListType() corev1alpha1.ObjectList {
	return &UserList{}
}

func (u *User) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &u.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func (UserList) ObjectType() corev1alpha1.Object {
	return &User{}
}
func (l *UserList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *UserList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*UserList))
}

func (l *UserList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *UserList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []User{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*User)))
	}
}
