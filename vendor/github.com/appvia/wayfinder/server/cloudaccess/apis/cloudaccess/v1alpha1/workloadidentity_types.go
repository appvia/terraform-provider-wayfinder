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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

const WorkloadIdentityKind = "WorkloadIdentity"

func WorkloadIdentityGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(GroupVersion.String(), WorkloadIdentityKind)
}

func NewWorkloadIdentity(name, namespace string) *WorkloadIdentity {
	return &WorkloadIdentity{
		TypeMeta: metav1.TypeMeta{
			Kind:       WorkloadIdentityKind,
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// +kubebuilder:webhook:name=workloadidentities.cloudaccess.appvia.io,mutating=false,path=/validate/cloudaccess.appvia.io/workloadidentities,verbs=create;update;delete,groups="cloudaccess.appvia.io",resources=workloadidentity,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadIdentity represents an identity for a kubernetes workload in a specific cloud provider /
// cloud account
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=workloadidentity,scope=Namespaced,categories={wayfinder}
type WorkloadIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadIdentitySpec   `json:"spec,omitempty"`
	Status WorkloadIdentityStatus `json:"status,omitempty"`
}

func (WorkloadIdentity) APIPath() string {
	return "workloadidentities"
}
func (c *WorkloadIdentity) Clone() corev1alpha1.Object {
	return c.DeepCopy()
}
func (c *WorkloadIdentity) CloneInto(o corev1alpha1.Object) {
	c.DeepCopyInto(o.(*WorkloadIdentity))
}
func (WorkloadIdentity) ListType() corev1alpha1.ObjectList {
	return &WorkloadIdentityList{}
}

// GetCommonStatus implements CommonStatusAware for CloudAccount
func (c *WorkloadIdentity) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetNamespacedName returns a namespaced name for this account
func (c *WorkloadIdentity) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// GetStatus returns the current overall status
func (c WorkloadIdentity) GetStatus() corev1alpha1.Status {
	return c.Status.Status
}

type WorkloadIdentityRole string

const (
	// WorkloadIdentityRoleNone defines the "minimal" cloud permissions
	// - For AWS the identity IS a role which we will add simply sts:GetCallerIdentity
	// - For Azure no permissions are required
	WorkloadIdentityRoleNone WorkloadIdentityRole = "None"
	// WorkloadIdentityRoleExternalDNS defines the required permissions for ExternalDNS to function
	// in a given cloud
	WorkloadIdentityRoleExternalDNS WorkloadIdentityRole = "ExternalDNS"
	// WorkloadIdentityRoleExternalDNS defines the required permissions for CertManager to function
	// in a given cloud
	WorkloadIdentityRoleCertManager WorkloadIdentityRole = "CertManager"
	// WorkloadIdentityRoleClusterAutoscaler defines the required permissions for the cluster
	// autoscaler to function in a given cloud (only needed on AWS)
	WorkloadIdentityRoleClusterAutoscaler WorkloadIdentityRole = "ClusterAutoscaler"
)

func (c WorkloadIdentityRole) String() string {
	return string(c)
}

// Valid return if WorkloadIdentityRole is valid.
// TODO: This should go away once we refactor cloudRoles
func (c WorkloadIdentityRole) Valid() error {
	switch strings.ToLower(c.String()) {
	case strings.ToLower(WorkloadIdentityRoleExternalDNS.String()),
		strings.ToLower(WorkloadIdentityRoleClusterAutoscaler.String()),
		strings.ToLower(WorkloadIdentityRoleCertManager.String()),
		strings.ToLower(WorkloadIdentityRoleNone.String()):
		return nil
	}
	return fmt.Errorf("unknown workload identity role: %s", c.String())
}

// WorkloadIdentitySpec defines the specification of a workload identity which should be provisioned
// +k8s:openapi-gen=true
type WorkloadIdentitySpec struct {
	// Cloud defines which cloud provider this workload identity is for
	// +kubebuilder:validation:Enum=aws;gcp;azure
	// +kubebuilder:validation:Required
	Cloud string `json:"cloud"`
	// CloudAccount defines which cloud account to build this workload identity in
	// +kubebuilder:validation:Required
	CloudAccount CloudAccountReference `json:"cloudAccount"`
	// Cluster is a reference to the cluster which this workload identity will be used in.
	// +kubebuilder:validation:Required
	Cluster corev1alpha1.Ownership `json:"cluster"`
	// ClusterServiceAccount is the name and namespace of the service account which will use this
	// identity in the target cluster. Required on AWS and GCP, optional (and unused) on Azure at
	// this time.
	// +kubebuilder:validation:Optional
	ClusterServiceAccount *ClusterServiceAccount `json:"clusterServiceAccount,omitempty"`
	// ProviderDetails provides additional fields which can be used for cloud-provider specific
	// data needed to provision a workload identity
	// +kubebuilder:validation:Optional
	ProviderDetails *WorkloadIdentityProviderDetails `json:"providerDetails,omitempty"`
	// Role must be the name of a valid workload identity role known to Wayfinder
	// Can optionally be None to indicate that no specific permissions are defined with the identity
	// +kubebuilder:validation:Required
	Role WorkloadIdentityRole `json:"role"`
	// IdentityOnly will create an identity associated with a cluster with no specific permissions
	// Must specify Role=None if this is true.
	// In AWS:
	//   - An IAM role is created and associated with a specific Kubernetes service account
	//   - no inline or attached policies are managed (post creation of the IAM role)
	//   - It is a "user" responsibility to attach policies to the IAM role
	// In Azure:
	//   - The user defined managed identity is created
	//   - No role definitions or role assignments are created
	//   - It is a "user" responsibility to create relevant role assignments
	// +kubebuilder:validation:Optional
	IdentityOnly bool `json:"identityOnly,omitempty"`
	// RoleParameters are any parameters required for the specified role
	// +kubebuilder:validation:Optional
	RoleParameters map[string]string `json:"roleParameters,omitempty"`
	// CloudResourceName specifies the name of the workload identity in the cloudaccount
	// Can be left blank so that the name is derived from the cluster name + resource name
	// +kubebuilder:validation:Optional
	CloudResourceName string `json:"cloudResourceName,omitempty"`
}

// ClusterServiceAccount represents the identity inside the cluster that will use the workload
// identity
type ClusterServiceAccount struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// WorkloadIdentityType represents the concrete type of a workload identity to provide
// +kubebuilder:validation:Enum=AWS;Azure;GCP
type WorkloadIdentityType string

const (
	// WorkloadIdentityTypeAWS is for AWS managed workload identity
	WorkloadIdentityTypeAWS WorkloadIdentityType = "AWS"
	// WorkloadIdentityTypeAzure is for Azure managed workload identity
	WorkloadIdentityTypeAzure WorkloadIdentityType = "Azure"
	// WorkloadIdentityTypeGCP is for GCP managed workload identity
	WorkloadIdentityTypeGCP WorkloadIdentityType = "GCP"
)

// WorkloadIdentityProviderDetails provides parameters that are specific to a particular type of
// workload identity
// +union
type WorkloadIdentityProviderDetails struct {
	// +unionDiscriminator
	// +kubebuilder:validation:Required
	// +required
	Type WorkloadIdentityType `json:"type"`

	// AWS holds parameters specific to AWS workload identities. Present only if
	// type is AWS.
	// +optional
	AWS *AWSWorkloadIdentityParameters `json:"aws,omitempty"`

	// GCP holds parameters specific to GCP workload identity. Present only if
	// type is GCP.
	// +optional
	GCP *GCPWorkloadIdentityParameters `json:"gcp,omitempty"`

	// Azure holds parameters specific to Azure workload identity. Present only if
	// type is Azure.
	// +optional
	Azure *AzureWorkloadIdentityParameters `json:"azure,omitempty"`
}

// AWSWorkloadIdentityParameters is the parameters for an AWS workload identity
type AWSWorkloadIdentityParameters struct{}

// GCPWorkloadIdentityParameters is the parameters for a GCP workload identity
type GCPWorkloadIdentityParameters struct{}

// AzureWorkloadIdentityParameters is the parameters for an Azure workload identity
type AzureWorkloadIdentityParameters struct {
	// PodSelector specifies the unique string that will be set on the Identity
	// This allows AAD Pod Identity to find the pods that are associated with it.
	// +kubebuilder:validation:Optional
	PodSelector string `json:"podSelector,omitempty"`
}

// WorkloadIdentityStatus defines the status of a cloud account
// +k8s:openapi-gen=true
type WorkloadIdentityStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
	// Identity contains a cloud-provider specific reference to the identity created for this
	// resource, e.g. an AWS ARN or GCP service account email
	Identity string `json:"identity,omitempty"`
}

// CloudResourceName returns the cloud provider name for this workload identity
func (c *WorkloadIdentity) CloudResourceName() string {
	if c.Spec.CloudResourceName != "" {
		return c.Spec.CloudResourceName
	}
	return fmt.Sprintf("%s-%s", c.Namespace, c.Name)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkloadIdentityList provides a list of workload identities
type WorkloadIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadIdentity `json:"items"`
}

func (WorkloadIdentityList) ObjectType() corev1alpha1.Object {
	return &WorkloadIdentity{}
}
func (l *WorkloadIdentityList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *WorkloadIdentityList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*WorkloadIdentityList))
}

func (l *WorkloadIdentityList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

func (l *WorkloadIdentityList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []WorkloadIdentity{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*WorkloadIdentity)))
	}
}

// IsReady checks if WorkloadIdentity is ready
func (c *WorkloadIdentity) IsReady() bool {
	return c.Status.InCondition(corev1alpha1.ConditionReady)
}
