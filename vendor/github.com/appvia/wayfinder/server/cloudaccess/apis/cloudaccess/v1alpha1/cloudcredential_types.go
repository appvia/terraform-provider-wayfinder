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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const CloudCredentialKind = "CloudCredential"

func CloudCredentialGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), CloudCredentialKind)
}

// +kubebuilder:webhook:name=cloudcredentials.cloudaccess.appvia.io,mutating=false,path=/validate/cloudaccess.appvia.io/cloudcredentials,verbs=create;update;delete,groups="cloudaccess.appvia.io",resources=cloudcredential,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=cloudcredentials.cloudaccess.appvia.io,mutating=true,path=/mutate/cloudaccess.appvia.io/cloudcredentials,verbs=create;update,groups="cloudaccess.appvia.io",resources=cloudcredential,versions=v1alpha1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudCredential represents a set of credentials to access a cloud account
// which Wayfinder can use to perform its operations
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cloudcredential,scope=Namespaced,categories={wayfinder}
type CloudCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudCredentialSpec   `json:"spec,omitempty"`
	Status CloudCredentialStatus `json:"status,omitempty"`
}

func (CloudCredential) APIPath() string {
	return "cloudcredentials"
}
func (c *CloudCredential) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *CloudCredential) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*CloudCredential))
}
func (CloudCredential) ListType() corev1alpha1.ObjectList {
	return &CloudCredentialList{}
}

// GetStatus returns the current overall status
func (c CloudCredential) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

// GetNamespacedName returns a namespaced name for this credential
func (c CloudCredential) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetCommonStatus implements CommonStatusAware for CloudCredential
func (c *CloudCredential) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// CloudCredentialSpec defines the metadata about the credentials with a reference to
// the kubernetes secret containing the credentials
// +k8s:openapi-gen=true
type CloudCredentialSpec struct {
	// Name is a human-understandable name for this credential
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Cloud defines which cloud provider this credential is for
	// +kubebuilder:validation:Enum=aws;gcp;azure
	// +kubebuilder:validation:Required
	Cloud string `json:"cloud"`
	// ImplicitIdentity specifies that any credentials are provided by the run time process environment
	// and NOT a secret reference. Typically this means that workload identity is to be used.
	// +kubebuilder:validation:Optional
	ImplicitIdentity bool `json:"implicitIdentity"`
	// ImplicitIdentityID specifies any ID that the run time process environment needs to authenticate
	// to a specific identity where more than one can be assigned to a process
	// +kubebuilder:validation:Optional
	ImplicitIdentityID string `json:"implicitIdentityID"`
	// CredentialsInputData can be used to populate the secret when creating/updating a
	// credential. This will never be populated when the credential is returned from the API.
	//
	// If specified, this must include the correct set of keys for credentials for the cloud
	// provider that CloudAccount references.
	// +kubebuilder:validation:Optional
	CredentialsInputData map[string]string `json:"credentialsInputData,omitempty"`
	// SecretRef is a reference to the Kubernetes secret containing the actual key data for this
	// credential. If the secret does not exist but CredentialsInputData is populated, this secret
	// will be created. This can also be a reference to an existing secret managed outside Wayfinder.
	//
	// Where CredentialsInputData is specified but this is left blank, Wayfinder will assign this value.
	// +kubebuilder:validation:Optional
	SecretRef *v1.SecretReference `json:"secretRef"`
	// CredentialsUpdated should be set to the current time when an underlying secret is
	// updated. This will be automatically set to the current time if CredentialsInputData
	// is set. If you manually change the secret outside Wayfinder, update this field to trigger
	// re-verification of this credential.
	CredentialsUpdated metav1.Time `json:"credentialsUpdated,omitempty"`
}

// UseImplicitIdentity returns true if this reference indicates that the process environment (e.g. workload identity) should be used.
func (r CloudCredentialSpec) UseImplicitIdentity() bool {
	return r.ImplicitIdentity
}

// CloudCredentialReference is a reference specifically to a cloud credential
// +k8s:openapi-gen=true
type CloudCredentialReference struct {
	// Namespace for the credential, specify empty for implicit credentials
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace"`
	// Name for the credential, specify empty for implicit credentials
	// +kubebuilder:validation:Optional
	Name string `json:"name"`
}

// UseParentIdentity returns true if this reference indicates that parent-account identity should be used.
func (r CloudCredentialReference) UseParentIdentity() bool {
	return r.Name == "" && r.Namespace == ""
}

// GetNamespacedName converts the CloudCredentialReference to a standard NamespacedName
func (r CloudCredentialReference) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName(r)
}

func (r CloudCredentialReference) GetOwnership() corev1alpha1.Ownership {
	return corev1alpha1.Ownership{
		Group:     GroupVersion.Group,
		Version:   GroupVersion.Version,
		Kind:      "CloudCredential",
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func GetCloudCredentialReference(o corev1alpha1.Ownership) CloudCredentialReference {
	return CloudCredentialReference{
		Namespace: o.Namespace,
		Name:      o.Name,
	}
}

// GetReference returns a reference to this cloud account.
func (c CloudCredential) GetReference() CloudCredentialReference {
	return CloudCredentialReference{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// CloudCredentialStatus represents the status of a set of credentials for cloud
// account access
type CloudCredentialStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Verified checks that the credentials are ok and valid
	Verified *bool `json:"verified,omitempty"`
	// Identity is the unique reference to the cloud principle e.g. aws role, gcp service-account etc.
	Identity string `json:"identity,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudCredentialList provides a list of cloud accounts
type CloudCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudCredential `json:"items"`
}

func (CloudCredentialList) ObjectType() corev1alpha1.Object {
	return &CloudCredential{}
}
func (l *CloudCredentialList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *CloudCredentialList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*CloudCredentialList))
}

func (l *CloudCredentialList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *CloudCredentialList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []CloudCredential{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*CloudCredential)))
	}
}
