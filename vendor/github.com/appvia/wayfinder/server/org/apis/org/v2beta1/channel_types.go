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

// +kubebuilder:webhook:name=channels.org.appvia.io,mutating=false,path=/validate/org.appvia.io/channels,verbs=create;update;delete,groups="org.appvia.io",resources=channels,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=channels.org.appvia.io,mutating=true,path=/mutate/org.appvia.io/channels,verbs=create;update,groups="org.appvia.io",resources=channels,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// ChannelKind is the resource crd name
const ChannelKind = "Channel"

// ChannelGVK is the GVK for channels
var ChannelGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    ChannelKind,
}

// NewChannel creates and returns an empty channel
func NewChannel(name string) *Channel {
	return &Channel{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       ChannelKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Channel is the Schema for the channels API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=channels,scope=Cluster,categories={wayfinder}
type Channel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChannelSpec   `json:"spec,omitempty"`
	Status ChannelStatus `json:"status,omitempty"`
}

func (Channel) GroupVersionKind() schema.GroupVersionKind {
	return ChannelGVK
}

func (Channel) APIPath() string {
	return "channels"
}

func (s *Channel) Clone() corev1alpha1.Object {
	return s.DeepCopy()
}

func (s *Channel) CloneInto(o corev1alpha1.Object) {
	s.DeepCopyInto(o.(*Channel))
}

func (Channel) ListType() corev1alpha1.ObjectList {
	return &ChannelList{}
}

func (s *Channel) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &s.Status.CommonStatus
}

// ChannelSpec defines the desired state of Channel
// +k8s:openapi-gen=true
type ChannelSpec struct {
	// DisplayName provides a user-friendly name for the channel
	// +kubebuilder:validation:Optional
	DisplayName string `json:"displayName"`
	// Description provides a summary of this channel
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// KubernetesVersions specifies the semver versions of Kubernetes by provider
	// The key of the provider relates to the cluster type (e.g. AKS, GKE, EKS)
	// - This is only required if the channel is the source of truth for Kubernetes versions
	// - If a provider (or all providers is absent) then the platform defaults will be used
	// +kubebuilder:validation:Optional
	KubernetesVersions map[string]string `json:"kubernetesVersions,omitempty"`
}

type PackageVersion struct {
	// Version is the version of the package
	Version string `json:"version,omitempty"`
	// Name is the metadata name of the global helm package
	Name string `json:"name,omitempty"`
}

// Versions are versions for a provider
// TODO: include other things that may need to be tied to a provider
type Versions struct {
	// Kubernetes is the default version of Kubernetes for this channel. Where KubernetesByRegion
	// is populated for the relevant region, that should be used in preference.
	// +kubebuilder:validation:Optional
	Kubernetes string `json:"kubernetes,omitempty"`

	// KubernetesByRegion provides cloud provider region-specific versions of Kubernetes. On Azure
	// and GCP, supported Kubernetes versions can vary between regions. If no value is populated
	// here, the value in Kubernetes should be used instead.
	// +kubebuilder:validation:Optional
	KubernetesByRegion map[string]string `json:"kubernetesByRegion,omitempty"`

	// Packages is a list of the packages versions on the channel
	// +kubebuilder:validation:Optional
	Packages map[string]string `json:"packages,omitempty"`
}

// ProviderVersions is a particular set of versions to use for a provider
// The key of the provider relates to the cluster type (e.g. AKS, GKE, EKS)
type ProviderVersions map[string]Versions

// ChannelStatus defines the observed state of Channel
// +k8s:openapi-gen=true
type ChannelStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`

	// Versions is a map of versions per provider...
	// +kubebuilder:validation:Optional
	Versions ProviderVersions `json:"versions,omitempty"`
}

// ChannelList contains a list of Channel
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Channel `json:"items"`
}

func (ChannelList) ObjectType() corev1alpha1.Object {
	return &Channel{}
}

func (l *ChannelList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}

func (l *ChannelList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ChannelList))
}

func (l *ChannelList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ChannelList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []Channel{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*Channel)))
	}
}
