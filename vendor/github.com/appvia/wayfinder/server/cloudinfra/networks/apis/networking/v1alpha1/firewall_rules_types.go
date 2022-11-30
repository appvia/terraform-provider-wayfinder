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

	corev1alpha1 "github.com/appvia/wayfinder/pkg/apis/core/v1alpha1"
)

// FirewallRulesKind is the kind for a Firewall
const FirewallRulesKind = "FirewallRules"

// FirewallRulesGVK is the GVK for a Firewall
var FirewallRulesGVK = schema.GroupVersionKind{
	Group:   GroupVersion.Group,
	Version: GroupVersion.Version,
	Kind:    FirewallRulesKind,
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=firewallrules
type FirewallRules struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FirewallSpec   `json:"spec,omitempty"`
	Status FirewallStatus `json:"status,omitempty"`
}

func (FirewallRules) APIPath() string {
	return "firewallrules"
}

// FirewallSpec defines the desired state of a firewall
// +k8s:openapi-gen=true
type FirewallSpec struct {
	// NetworkRef is a reference to the network associated with the firewall.
	// +kubebuilder:validation:Required
	NetworkRef corev1alpha1.Ownership `json:"networkRef,omitempty"`
	// Rules is a list of firewall rules.
	// +kubebuilder:validation:Required
	Rules []FirewallRule `json:"rules"`
}

// Port is a network port.
// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=65535
type Port uint16

// FirewallRule represents the various options associated with a firewall rule.
// Depending on the cloud, a single FirewallRule might be expanded to multiple
// individual firewall rules.
type FirewallRule struct {
	// Name is the name of the firewall rule.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is an optional description of the firewall rule.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Action dictates whether to allow or deny matching traffic.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Allow;Deny
	Action string `json:"action"`
	// CIDRBlocks is the list of IP address ranges that this rule applies to.
	// +kubebuilder:validation:Optional
	CIDRBlocks []IPv4CIDR `json:"cidrBlocks"`
	// Direction dictates whether this rule applies to inbound or outbound traffic.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Ingress;Egress
	Direction string `json:"direction"`
	// IPVersion is the version of the Internet Protocol for the firewall rule.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=4;6
	// +kubebuilder:default:4
	IPVersion uint16 `json:"ipVersion,omitempty"`
	// Ports is a list of protocols that this firewall rule applies to.
	// +kubebuilder:validation:Required
	Protocols []string `json:"protocols,omitempty"`
	// Ports is a list of port numbers that this firewall rule applies to.
	// If omitted, the rule applies to all ports.
	// +kubebuilder:validation:Optional
	Ports []Port `json:"ports,omitempty"`
	// Priority dictates the precedences of the firewall rule. Lower values
	// indicate higher priorities.
	// +kubebuilder:validation:Optional
	Priority uint16 `json:"priority,omitempty"`

	// TODO: Investigate the implication of tags across clouds, e.g. in GCP
	// tags are used to *target* VMs for the firewall rule, whereas on AWS
	// tags are for organisational purposes and are meaningless.
}

// FirewallStatus defines the observed state of a firewall
// +k8s:openapi-gen=true
type FirewallStatus struct{}
