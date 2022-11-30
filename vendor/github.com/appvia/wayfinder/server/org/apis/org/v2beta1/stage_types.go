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

// +kubebuilder:webhook:name=stages.org.appvia.io,mutating=false,path=/validate/org.appvia.io/stages,verbs=create;update;delete,groups="org.appvia.io",resources=stages,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=stages.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/stages,verbs=create;update,groups="org.appvia.io",resources=stages,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// StageKind is the resource crd name
const StageKind = "Stage"

// StageGVK is the GVK for stages
var StageGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    StageKind,
}

// NewStage creates and returns an empty stage
func NewStage(name string) *Stage {
	return &Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       StageKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// StageSpec defines the desired state of Stage
// +k8s:openapi-gen=true
type StageSpec struct {
	// DisplayName provides a user-friendly name for the stage
	// +kubebuilder:validation:Optional
	DisplayName string `json:"displayName"`
	// Description provides a summary of this stage
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Description string `json:"description"`
}

// StageStatus defines the observed state of Stage
// +k8s:openapi-gen=true
type StageStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stage is the Schema for the stages API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=stages,scope=Cluster,categories={wayfinder}
type Stage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StageSpec   `json:"spec,omitempty"`
	Status StageStatus `json:"status,omitempty"`
}

func (Stage) APIPath() string {
	return "stages"
}
func (s *Stage) Clone() corev1alpha1.Object {
	return s.DeepCopy()
}
func (s *Stage) CloneInto(o corev1alpha1.Object) {
	s.DeepCopyInto(o.(*Stage))
}
func (Stage) ListType() corev1alpha1.ObjectList {
	return &StageList{}
}

func (s *Stage) GetCommonStatus() *corev1alpha1.CommonStatus {
	s.Status.SetStatusIfNotSet(corev1alpha1.SuccessStatus)
	return &s.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StageList contains a list of Stage
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stage `json:"items"`
}

func (StageList) ObjectType() corev1alpha1.Object {
	return &Stage{}
}
func (l *StageList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *StageList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*StageList))
}

func (l *StageList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *StageList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Stage{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Stage)))
	}
}
