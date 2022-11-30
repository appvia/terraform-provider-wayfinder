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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// QuotaLimitKind is the kind for a QuotaLimit
const QuotaLimitKind = "QuotaLimit"

// QuotaLimitGVK is the GVK for a QuotaLimit
var QuotaLimitGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    QuotaLimitKind,
}

// NewQuotaLimit creates a minimal QuotaLimit object with no spec
func NewQuotaLimit(name, namespace string) *QuotaLimit {
	return &QuotaLimit{
		TypeMeta: metav1.TypeMeta{
			Kind:       QuotaLimitKind,
			APIVersion: GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// +kubebuilder:webhook:name=quotalimits.compute.appvia.io,mutating=false,path=/validate/compute.appvia.io/quotalimits,verbs=create;update;delete,groups="compute.appvia.io",resources=quotalimits,versions=v2beta1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=quotalimits.compute.appvia.io,mutating=true,path=/mutate/compute.appvia.io/quotalimits,verbs=create;update,groups="compute.appvia.io",resources=quotalimits,versions=v2beta1,failurePolicy=fail,sideEffects=NoneOnDryRun,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuotaLimit is the schema for QuotaLimit
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=quotalimits,scope=Namespaced,categories={wayfinder}
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type QuotaLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaLimitSpec   `json:"spec,omitempty"`
	Status QuotaLimitStatus `json:"status,omitempty"`
}

func (QuotaLimit) APIPath() string {
	return "quotalimits"
}
func (ql *QuotaLimit) Clone() corev1alpha1.Object {
	return ql.DeepCopy()
}
func (ql *QuotaLimit) CloneInto(o corev1alpha1.Object) {
	ql.DeepCopyInto(o.(*QuotaLimit))
}
func (QuotaLimit) ListType() corev1alpha1.ObjectList {
	return &QuotaLimitList{}
}

// ResourceList is a strongly typed wrapper for simple conversion to k8s ResourceLists objects
// Avoids swagger errors when using native k8s ResourceList - object has no key "inf.Scale"
type ResourceList map[string]string

// ToNativeResourceList will convert to a k8s native ResourseList object
func (r ResourceList) ToNativeResourceList() (v1.ResourceList, error) {
	rl := make(v1.ResourceList)
	for k, v := range r {
		if res, err := resource.ParseQuantity(v); err != nil {
			return nil, err
		} else {
			// TODO: verify the key is for a valid resource
			resName := v1.ResourceName(k)
			rl[resName] = res
		}
	}
	return rl, nil
}

// LimitRangeDefault provides defaults for pod container and volumes
// Managed by a tenant with understanding of the workloads involved
type LimitRangeDefault struct {
	// Type of resource that this limit applies to.
	Type v1.LimitType `json:"type"`
	// Default resource requirement limit value by resource name if resource limit is omitted.
	// +optional
	Default ResourceList `json:"default,omitempty"`
	// DefaultRequest is the default resource requirement request value by resource name if resource request is omitted.
	// +optional
	DefaultRequest ResourceList `json:"defaultRequest,omitempty"`
}

// ToNativeResource will convert a LimitRangeConstraint to a native LimitRangeItem
func (c LimitRangeDefault) ToNativeResource() (v1.LimitRangeItem, error) {
	lri := v1.LimitRangeItem{}
	def, err := c.Default.ToNativeResourceList()
	if err != nil {
		return lri, err
	}

	defReq, err := c.DefaultRequest.ToNativeResourceList()
	if err != nil {
		return lri, err
	}
	lri.Default = def
	lri.DefaultRequest = defReq
	lri.Type = c.Type
	return lri, nil
}

// LimitRangeConstraint provides constraining Limits to be applied to a managed namespace
// Managed by tenant
type LimitRangeConstraint struct {
	// Type of resource that this limit applies to.
	Type v1.LimitType `json:"type"`
	// Max usage constraints on this kind by resource name.
	// +optional
	Max ResourceList `json:"max,omitempty"`
	// MaxLimitRequestRatio if specified, the named resource must have a request and limit that are both non-zero where limit divided by request is less than or equal to the enumerated value; this represents the max burst for the named resource.
	// +optional
	MaxLimitRequestRatio ResourceList `json:"maxLimitRequestRatio,omitempty"`
}

// ToNativeResource will convert a LimitRangeConstraint to a native LimitRangeItem
func (c LimitRangeConstraint) ToNativeResource() (v1.LimitRangeItem, error) {
	lri := v1.LimitRangeItem{}

	max, err := c.Max.ToNativeResourceList()
	if err != nil {
		return lri, err
	}

	maxLimitRequestRatio, err := c.MaxLimitRequestRatio.ToNativeResourceList()
	if err != nil {
		return lri, err
	}

	lri.Type = c.Type
	lri.Max = max
	lri.MaxLimitRequestRatio = maxLimitRequestRatio
	return lri, nil
}

// Constraints controlling a namespace by an owner
type Constraints struct {
	// QuotaHard are applied as ResourceQuota objects to all namespaces for a tenant of that type
	// There are the maximun resource usage in total for a namespace e.g.:
	// hardQuota:
	//   requests.cpu: "1"
	//   requests.memory: "1Gi"
	//   limit.cpu: "2"
	// +kubebuilder:validation:Optional
	HardQuota ResourceList `json:"hardQuota,omitempty"`
	// LimitRanges (in the context of constraints) are applied as LimitRanges (using Max)
	// These are the largest values that can be used by a single resource in a namespace
	// They will prevent workloads from using up all resources on a single node
	// +kubebuilder:validation:Optional
	LimitRanges []LimitRangeConstraint `json:"limitRanges"`
}

// ToLimitRangeResources will convert a LimitRangeConstraint to a native LimitRangeItems list
func (c Constraints) ToLimitRangeResources() ([]v1.LimitRangeItem, error) {
	lrItems := []v1.LimitRangeItem{}
	for _, lri := range c.LimitRanges {
		lr, err := lri.ToNativeResource()
		if err != nil {
			return lrItems, err
		}
		lrItems = append(lrItems, lr)
	}
	return lrItems, nil
}

// QuotaLimitTemplate provides template values on clusters and plans
type QuotaLimitTemplate struct {
	// TemplateName is the immutable name to track the tempate that was used
	// Would normally specify a QuotaLimit "size" for a tenant
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Constraints are for providing management of resource constraints to workloads
	// - Typically only managed by cluster owners to provide hard resource constraints
	// - Used to prevent noisy neigbor issues in a multi-tenanted environment
	// +kubebuilder:validation:Optional
	Constraints *Constraints `json:"constraints,omitempty"`
	// ResourceDefaults provide defaults for workloads (to prevent initial constraint violations)
	// Not set directly or managed by owners (as only maximum values taken so should not conflict)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ResourceDefaults []LimitRangeDefault `json:"resourceDefaults,omitempty"`
}

// QuotaLimitSpec provide the default ResourceQuota and LimitRange settings.
// They can be applied to all namespaces for a workspace
// Optionally they can relate to specific namespaces only
type QuotaLimitSpec struct {
	// TemplateName is the immutable name to track the tempate that was used
	// Would normally specify a QuotaLimit "size" for a tenant
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Constraints are for providing management of resource constraints to workloads
	// - Typically only managed by cluster owners to provide hard resource constraints
	// - Used to prevent noisy neigbor issues in a multi-tenanted environment
	// +kubebuilder:validation:Optional
	Constraints *Constraints `json:"constraints,omitempty"`
	// NamespaceClaim is the reference for a specific namespace claim that resulted in this object
	// +kubebuilder:validation:Required
	NamespaceClaim corev1alpha1.Ownership `json:"namespaceClaim"`
}

// QuotaLimitStatus provides the status of a QuotaLimit object
// +k8s:openapi-gen=true
type QuotaLimitStatus struct {
	// Status is the status of the QuotaLimit
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus implements CommonStatusAware for QuotaLimits
func (ql *QuotaLimit) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &ql.Status.CommonStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuotaLimitList contains a list of quotalimit objects
type QuotaLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuotaLimit `json:"items"`
}

func (QuotaLimitList) ObjectType() corev1alpha1.Object {
	return &QuotaLimit{}
}
func (l *QuotaLimitList) Clone() corev1alpha1.ObjectList {
	return l.DeepCopy()
}
func (l *QuotaLimitList) CloneInto(o corev1alpha1.ObjectList) {
	l.DeepCopyInto(o.(*QuotaLimitList))
}

// GetItems unpacks the list into client.Objects
func (l *QuotaLimitList) GetItems() []corev1alpha1.Object {
	res := make([]corev1alpha1.Object, len(l.Items))
	for i, item := range l.Items {
		ref := item
		res[i] = &ref
	}
	return res
}

// SetItems adds untyped QuotaLimit items to the current list
func (l *QuotaLimitList) SetItems(objects []corev1alpha1.Object) {
	l.Items = []QuotaLimit{}
	for _, o := range objects {
		l.Items = append(l.Items, *(o.(*QuotaLimit)))
	}
}
