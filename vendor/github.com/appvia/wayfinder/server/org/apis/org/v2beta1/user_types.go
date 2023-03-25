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

// +kubebuilder:webhook:name=users.org.appvia.io,mutating=false,path=/validate/org.appvia.io/users,verbs=create;update;delete,groups="org.appvia.io",resources=users,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=users.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/users,verbs=create;update,groups="org.appvia.io",resources=users,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// UserKind is the kind for a human user of Wayfinder
const UserKind = "User"

// UserGVK is the GVK for a User
var UserGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    UserKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User is the Schema for the users API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=users,scope=Cluster,categories={wayfinder}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

func (User) GroupVersionKind() schema.GroupVersionKind {
	return UserGVK
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
	u.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &u.Status.CommonStatus
}

func (u *User) ToSubject() Subject {
	return Subject{
		Kind: SubjectKindUser,
		Name: u.Name,
	}
}

// UserSpec defines the desired state of User
// +k8s:openapi-gen=true
type UserSpec struct {
	// Username is the userame or identity for this user - typically this would be an email address.
	// It must be unique.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Username string `json:"username"`
	// Disabled indicates if the user is disabled
	Disabled bool `json:"disabled"`
}

// UserStatus defines the observed state of User
// +k8s:openapi-gen=true
type UserStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// UserList contains a list of User
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
