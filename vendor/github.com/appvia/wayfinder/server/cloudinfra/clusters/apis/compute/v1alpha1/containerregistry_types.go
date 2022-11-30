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
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const (
	// ConditionContainerRegistryExists indicates that the underlying registry has
	// been created
	ConditionContainerRegistryExists corev1alpha1.ConditionType = "ContainerRegistryExists"

	// RegistryProviderUnmanaged represents an unmanaged registry
	RegistryProviderUnmanaged = "unmanaged"
)

// +kubebuilder:webhook:name=containerregistries.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/containerregistries,verbs=create;update;delete,groups="compute.appvia.io",resources=containerregistries,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerRegistry is the container registry type
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=containerregistries,scope=Namespaced,categories={wayfinder}
type ContainerRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerRegistrySpec   `json:"spec,omitempty"`
	Status ContainerRegistryStatus `json:"status,omitempty"`
}

func (ContainerRegistry) APIPath() string {
	return "containerregistries"
}
func (c *ContainerRegistry) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *ContainerRegistry) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*ContainerRegistry))
}

func (ContainerRegistry) ListType() corev1alpha1.ObjectList {
	return &ContainerRegistryList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerRegistryList provides a list of container registries
type ContainerRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerRegistry `json:"items"`
}

func (ContainerRegistryList) ObjectType() corev1alpha1.Object {
	return &ContainerRegistry{}
}
func (l *ContainerRegistryList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *ContainerRegistryList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*ContainerRegistryList))
}

func (l *ContainerRegistryList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *ContainerRegistryList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []ContainerRegistry{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*ContainerRegistry)))
	}
}

// +kubebuilder:webhook:name=globalcontainerregistries.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/globalcontainerregistries,verbs=create;update;delete,groups="compute.appvia.io",resources=globalcontainerregistries,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalContainerRegistry is the global container registry type
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=globalcontainerregistries,scope=Cluster,categories={wayfinder}
type GlobalContainerRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerRegistrySpec   `json:"spec,omitempty"`
	Status ContainerRegistryStatus `json:"status,omitempty"`
}

func (GlobalContainerRegistry) APIPath() string {
	return "globalcontainerregistries"
}
func (c *GlobalContainerRegistry) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *GlobalContainerRegistry) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*GlobalContainerRegistry))
}
func (GlobalContainerRegistry) ListType() corev1alpha1.ObjectList {
	return &GlobalContainerRegistryList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalContainerRegistryList provides a list of global container registries
type GlobalContainerRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalContainerRegistry `json:"items"`
}

func (GlobalContainerRegistryList) ObjectType() corev1alpha1.Object {
	return &GlobalContainerRegistry{}
}
func (l *GlobalContainerRegistryList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *GlobalContainerRegistryList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*GlobalContainerRegistryList))
}

func (l *GlobalContainerRegistryList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *GlobalContainerRegistryList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []GlobalContainerRegistry{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*GlobalContainerRegistry)))
	}
}

// ContainerRegistryStatus defines the status of a container registry
// +k8s:openapi-gen=true
type ContainerRegistryStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// PullSecret is a reference to the imagePullSecret
	// +kubebuilder:validation:Optional
	PullSecret *v1.SecretReference `json:"pullSecret,omitempty"`
}

// GetCommonStatus implements CommonStatusAware
func (c *ContainerRegistry) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetCommonStatus implements CommonStatusAware
func (c *GlobalContainerRegistry) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// ContainerRegistrySpec defines the specification of a container registry
// +k8s:openapi-gen=true
type ContainerRegistrySpec struct {
	// Target contains targeting information for this registry
	// +kubebuilder:validation:Optional
	Target ContainerRegistryTarget `json:"target"`
	// Provider defines which provider to use to create this registry
	// +kubebuilder:validation:Enum=aws;gcp;azure;unmanaged
	// +kubebuilder:validation:Required
	Provider string `json:"provider"`
	// ProviderDetails provides additional fields which can be used for provider specific
	// data needed to provision this registry
	// +kubebuilder:validation:Optional
	ProviderDetails *ContainerRegistryProviderDetails `json:"providerDetails,omitempty"`
}

// GlobalOrScopedContainterRegistry provides a common interface for global or workspace-scoped Container Regigistries
// +kubebuilder:object:generate=false
type GlobalOrScopedContainerRegistry interface {
	corev1alpha1.Object
	IsGlobal() bool
	GetContainerRegistrySpec() *ContainerRegistrySpec
	GetContainerRegistryStatus() *ContainerRegistryStatus
	GetClusterTarget() *metav1.LabelSelector
	GetNamespaceTarget() *metav1.LabelSelector
}

func NewGlobalOrScopedContainerRegistry(ref types.NamespacedName) GlobalOrScopedContainerRegistry {
	if ref.Namespace == "" {
		return &GlobalContainerRegistry{
			ObjectMeta: metav1.ObjectMeta{
				Name: ref.Name,
			},
		}
	} else {
		return &ContainerRegistry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ref.Name,
				Namespace: ref.Namespace,
			},
		}
	}
}

// GetContainerRegistrySpec returns the ContainerRegistrySpec
func (c *ContainerRegistry) GetContainerRegistrySpec() *ContainerRegistrySpec {
	return &c.Spec
}

// GetContainerRegistrySpec returns the ContainerRegistrySpec
func (c *GlobalContainerRegistry) GetContainerRegistrySpec() *ContainerRegistrySpec {
	return &c.Spec
}

// GetContainerRegistryStatus returns the ContainerRegistryStatus
func (c *ContainerRegistry) GetContainerRegistryStatus() *ContainerRegistryStatus {
	return &c.Status
}

// GetContainerRegistryStatus returns the ContainerRegistryStatus
func (c *GlobalContainerRegistry) GetContainerRegistryStatus() *ContainerRegistryStatus {
	return &c.Status
}

func (c *ContainerRegistry) GetClusterTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Cluster
}

func (c *GlobalContainerRegistry) GetClusterTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Cluster
}

func (c *ContainerRegistry) GetNamespaceTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Namespace
}

func (c *GlobalContainerRegistry) GetNamespaceTarget() *metav1.LabelSelector {
	return &c.Spec.Target.Namespace
}

// IsGLobal
func (c *ContainerRegistry) IsGlobal() bool {
	return false
}

// IsGLobal
func (c *GlobalContainerRegistry) IsGlobal() bool {
	return true
}

// ContainerRegistryType represents the concrete type of a Container Registry to provide
// +kubebuilder:validation:Enum=Unmanaged
type ContainerRegistryType string

const (
	// ContainerRegistryTypeUnmanaged is the tyoe for unmanaged registries
	ContainerRegistryTypeUnmanaged ContainerRegistryType = "Unmanaged"
)

// ContainerRegistryProviderDetails provides parameters that are specific to a particular type of
// Container Registry
// +union
type ContainerRegistryProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type ContainerRegistryType `json:"type"`

	// Unmanaged holds parameters specific to an Unmanaged container registries.
	// +optional
	Unmanaged *UnmanagedContainerRegistryParameters `json:"unmanaged,omitempty"`
}

// UnmanagedContainerRegistryParameters
type UnmanagedContainerRegistryParameters struct {
	// +kubebuilder:validation:Optional
	Secret *v1.SecretReference `json:"secret,omitempty"`
}

// ContainerRegistryTarget is the targeting specification for this registry
// +k8s:openapi-gen=true
type ContainerRegistryTarget struct {
	// Cluster specifies the cluster(s) this registry should be made available to
	Cluster metav1.LabelSelector `json:"cluster,omitempty"`
	// Namespace specifies the Namespace(s) this registry should be made available to
	Namespace metav1.LabelSelector `json:"namespace,omitempty"`
}

var ContainerRegistryConditions = []corev1alpha1.ConditionSpec{
	{Type: corev1alpha1.ConditionReady, Name: "Registry Ready"},
	{Type: ConditionContainerRegistryExists, Name: "Registry exists"},
}
