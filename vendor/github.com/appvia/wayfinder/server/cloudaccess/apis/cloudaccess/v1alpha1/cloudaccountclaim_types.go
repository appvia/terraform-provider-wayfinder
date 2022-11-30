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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const CloudAccountClaimKind = "CloudAccountClaim"

func CloudAccountClaimGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), CloudAccountClaimKind)
}

// CloudAccountClaimSpec defines the desired state of CloudAccountClaim
// +k8s:openapi-gen=true
type CloudAccountClaimSpec struct {
	// Name is the name of the account to create. Either Name or Stage must be populated.
	// +kubebuilder:validation:Optional
	//
	// Populate name to choose a custom name, and populate stage to follow the account naming rules
	// for the parent.
	Name string `json:"name,omitempty"`
	// Stage is the lifecycle stage to create the cloud account for. Either Name or Stage must
	// be populated.
	//
	// Populate stage to following the account naming rules for the parent, and populate name to
	// choose a custom name.
	// +kubebuilder:validation:Optional
	Stage string `json:"stage,omitempty"`
	// Parent is a reference to a CloudAccount which this claim should use for provisioning the
	// account. If this is an Organization account, this will trigger the creation of a new account
	// within this parent. If it is a Shared or Wayfinder Managed account, it will be validated and used
	// directly. Any child account will inherit the Cloud provider from this parent.
	// +kubebuilder:validation:Required
	Parent CloudAccountReference `json:"parent"`
}

// CloudAccountClaimStatus defines the observed state of the provisioned account
// +k8s:openapi-gen=true
type CloudAccountClaimStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Name is the resulting cloud account name
	Name string `json:"name,omitempty"`
	// CloudAccountRef is a reference to the assigned or created cloud account
	CloudAccountRef *CloudAccountReference `json:"cloudAccountRef,omitempty"`
	// Identifier is the assigned account ID / project ID / subscription ID
	Identifier string `json:"identifier,omitempty"`
}

// +kubebuilder:webhook:name=cloudaccountclaims.cloudaccess.appvia.io,mutating=false,path=/validate/cloudaccess.appvia.io/cloudaccountclaims,verbs=create;update;delete,groups="cloudaccess.appvia.io",resources=cloudaccountclaims,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccountClaim represents a request for a cloud account to come into existence for a workspace
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cloudaccountclaims,scope=Namespaced,categories={wayfinder}
type CloudAccountClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudAccountClaimSpec   `json:"spec,omitempty"`
	Status CloudAccountClaimStatus `json:"status,omitempty"`
}

func (CloudAccountClaim) APIPath() string {
	return "cloudaccountclaims"
}
func (c *CloudAccountClaim) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *CloudAccountClaim) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*CloudAccountClaim))
}
func (CloudAccountClaim) ListType() corev1alpha1.ObjectList {
	return &CloudAccountClaimList{}
}

// GetCommonStatus implements CommonStatusAware for CloudAccountClaim
func (c *CloudAccountClaim) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetNamespacedName returns a namespaced name for this claim
func (c CloudAccountClaim) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudAccountClaimList contains a list of CloudAccountClaim
type CloudAccountClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudAccountClaim `json:"items"`
}

func (CloudAccountClaimList) ObjectType() corev1alpha1.Object {
	return &CloudAccountClaim{}
}

func (l *CloudAccountClaimList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *CloudAccountClaimList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*CloudAccountClaimList))
}
func (l *CloudAccountClaimList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *CloudAccountClaimList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []CloudAccountClaim{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*CloudAccountClaim)))
	}
}
