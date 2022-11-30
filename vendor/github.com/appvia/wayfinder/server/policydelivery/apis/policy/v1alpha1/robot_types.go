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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// AnnotationRobotAccount indicates we generate static token not exchange token
	AnnotationRobotAccount = "policies.appvia.io/robot-account"
)

// RobotSpec defines the desired state of policy
type RobotSpec struct {
	// Description provides a short summary on the use of the robot account
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// SecretRef is a reference to the underlying kubernetes secret
	// +kubebuilder:validation:Optional
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
}

// RobotStatus defines the observed state of status on a policy
// +k8s:openapi-gen=true
type RobotStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// DEPRECATED: secret reference is no longer in use
	// SecretRef is a reference to the underlying kubernetes secret
	// +kubebuilder:validation:Optional
	SecretRef *corev1alpha1.Ownership `json:"secretRef,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Robot is the Schema for the robot accounts API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=robots,scope=Namespaced,categories={wayfinder}
type Robot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RobotSpec   `json:"spec,omitempty"`
	Status RobotStatus `json:"status,omitempty"`
}

func (Robot) APIPath() string {
	return "robots"
}
func (r *Robot) Clone() corev1alpha1.Object {
	return r.DeepCopy()
}
func (r *Robot) CloneInto(o corev1alpha1.Object) {
	r.DeepCopyInto(o.(*Robot))
}
func (Robot) ListType() corev1alpha1.ObjectList {
	return &RobotList{}
}

func (r *Robot) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &r.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RobotList contains a list of robot accounts
type RobotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Robot `json:"items"`
}

func (RobotList) ObjectType() corev1alpha1.Object {
	return &Robot{}
}
func (l *RobotList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *RobotList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*RobotList))
}

func (l *RobotList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *RobotList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Robot{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Robot)))
	}
}
