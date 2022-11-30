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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// NamespaceClaimGVK is the GVK for a NamespaceClaim
var NamespaceClaimGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    NamespaceClaimKind,
}

const NamespaceClaimKind = "NamespaceClaim"

// NamespaceClaimSpec defines the desired state of NamespaceClaim
// +k8s:openapi-gen=true
type NamespaceClaimSpec struct {
	// Cluster is the cluster the namespace resides
	// +kubebuilder:validation:Required
	Cluster corev1alpha1.Ownership `json:"cluster"`
	// Name is the name of the namespace to create
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Annotations is a series of annotations on the namespace
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Labels is a series of labels for the namespace
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// ResourceDefaults provide defaults for workloads (to prevent initial constraint violations)
	// Not set directly or managed by owners (as only maximum values taken so should not conflict)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ResourceDefaults *[]LimitRangeDefault `json:"resourceDefaults,omitempty"`
}

// NamespaceClaimStatus defines the observed state of NamespaceClaim
// +k8s:openapi-gen=true
type NamespaceClaimStatus struct {
	// Status is the status of the namespace
	corev1alpha1.CommonStatus `json:",inline"`
	// AppliedConstraints details any applicable constraints applied to this namespace by the
	// cluster owner
	AppliedConstraints *Constraints `json:"appliedConstraints,omitempty"`
	// PackageValues are values present in packages that a user may need to construct manifests.
	// +kubebuilder:validation:MinItems=1
	PackageValues []PackageValue `json:"packageValues,omitempty"`
	// DNSZones are the zones associated with the namespace
	// +kubebuilder:validation:Optional
	DNSZones []string `json:"dnsZones"`
}

// PackageValue represents a value in a package release
// +k8s:openapi-gen=true
type PackageValue struct {
	// Name is the name of the value field
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// Description is an explanation of value's significance/usage
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// Value is a representation of the value
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:webhook:name=namespaceclaims.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/namespaceclaims,verbs=create;update;delete,groups="compute.appvia.io",resources=namespaceclaims,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=namespaceclaims.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/namespaceclaims,verbs=create;update,groups="compute.appvia.io",resources=namespaceclaims,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceClaim is the Schema for the namespaceclaims API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=namespaceclaims,scope=Namespaced,categories={wayfinder}
type NamespaceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceClaimSpec   `json:"spec,omitempty"`
	Status NamespaceClaimStatus `json:"status,omitempty"`
}

func (NamespaceClaim) APIPath() string {
	return "namespaceclaims"
}
func (n *NamespaceClaim) Clone() corev1alpha1.Object {
	return n.DeepCopy()
}
func (n *NamespaceClaim) CloneInto(o corev1alpha1.Object) {
	n.DeepCopyInto(o.(*NamespaceClaim))
}
func (NamespaceClaim) ListType() corev1alpha1.ObjectList {
	return &NamespaceClaimList{}
}

// Owner implements the Ownable interface
func (n NamespaceClaim) Owner() corev1alpha1.Ownership {
	return n.Spec.Cluster
}

// GetCommonStatus implements CommonStatusAware for namespace claims
func (n *NamespaceClaim) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &n.Status.CommonStatus
}

// ToLimitRangeDefaults will convert a ToLimitRangeDefaults List to a native LimitRangeItems list
func (n *NamespaceClaim) ToLimitRangeDefaults() ([]v1.LimitRangeItem, error) {
	lrItems := []v1.LimitRangeItem{}
	if n.Spec.ResourceDefaults == nil {
		return lrItems, nil
	}
	for _, lrd := range *n.Spec.ResourceDefaults {
		lr, err := lrd.ToNativeResource()
		if err != nil {
			return lrItems, err
		}
		lrItems = append(lrItems, lr)
	}
	return lrItems, nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceClaimList contains a list of NamespaceClaim
type NamespaceClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceClaim `json:"items"`
}

func (NamespaceClaimList) ObjectType() corev1alpha1.Object {
	return &NamespaceClaim{}
}
func (l *NamespaceClaimList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *NamespaceClaimList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*NamespaceClaimList))
}

func (l *NamespaceClaimList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *NamespaceClaimList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []NamespaceClaim{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*NamespaceClaim)))
	}
}
