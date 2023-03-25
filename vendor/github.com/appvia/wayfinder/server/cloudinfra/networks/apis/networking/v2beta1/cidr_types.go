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
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

// IPv4CIDRs is a list of IPv4 addresses
type IPv4CIDRs []IPv4CIDR

// Strings returns the cidr blocks as string slice
func (c IPv4CIDRs) Strings() []string {
	cidrBlocks := make([]string, len(c))
	for i, cidrBlock := range c {
		cidrBlocks[i] = cidrBlock.String()
	}
	return cidrBlocks
}

// IPv4CIDRBlocks is a list of IPv4 addresses with a type
type IPv4CIDRBlocks []IPv4CIDRBlock

func (c IPv4CIDRBlocks) GetNetworks() IPv4CIDRs {
	out := []IPv4CIDR{}
	for _, cidrBlock := range c {
		out = append(out, cidrBlock.CIDR)
	}
	return out
}

// IPv4CIDRs returns just the cidrs as a list of IPv4CIDR with optional matching types
// if no parameters, will return all cidr blocks
func (c IPv4CIDRBlocks) GetNetworksOfTypes(netTypes ...AssignableNetworkType) IPv4CIDRs {
	out := []IPv4CIDR{}

	for _, netType := range netTypes {
		out = append(out, c.GetNetworksOfType(netType)...)
	}

	return out
}

// GetNetworksOfType returns cidr blocks for the use case
func (c IPv4CIDRBlocks) GetNetworksOfType(netType AssignableNetworkType) IPv4CIDRs {
	out := []IPv4CIDR{}
	// see if we have a range matching specified type
	for _, cidrBlock := range c {
		if cidrBlock.Type == netType {
			// return the first relevant cidr block
			out = append(out, cidrBlock.CIDR)
		}
	}
	return out
}

// GetFirstNetworkOfType returns the first network of a given type
// TODO: I'd like this to return an error if not founc but first I want to make it work with what exists
func (c IPv4CIDRBlocks) GetFirstNetworkOfType(netType AssignableNetworkType) IPv4CIDR {
	if len(c.GetNetworksOfType(netType)) == 0 {
		return IPv4CIDR("")
	}
	return c.GetNetworksOfType(netType)[0]
}

// HasRequiredNetworks
func (c IPv4CIDRBlocks) HasRequiredNetworks(netTypes ...AssignableNetworkType) bool {
	for _, netType := range netTypes {
		if !c.HasNetworkOfType(netType) {
			return false
		}
	}
	return true
}

// HasNetworkOFType
func (c IPv4CIDRBlocks) HasNetworkOfType(netType AssignableNetworkType) bool {
	for _, cidrBlock := range c {
		if cidrBlock.Type == netType {
			return true
		}
	}
	return false
}

// CidrBlock represents an IPv4Cidr block and its use case
type IPv4CIDRBlock struct {
	// Type is what the cidr block is used for
	// +kubebuilder:validation:Required
	Type AssignableNetworkType `json:"type"`
	// CIDR is the cidr range in the A.B.C.D/N format
	// +kubebuilder:validation:Optional
	CIDR IPv4CIDR `json:"cidr"`
	// Size is the size of the CIDR range to be allocated if not specified
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=32
	Size int `json:"size"`
}

// IPv4CIDR represents an IP range in the A.B.C.D/N format
// +kubebuilder:validation:Pattern=`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}/([0-9]|[1-2][0-9]|3[0-2]))?$`
type IPv4CIDR string

// String returns the string representation of the cidr block
func (c IPv4CIDR) String() string {
	return string(c)
}

// Parse returns the parsed cidr range in ip, block or error
func (c IPv4CIDR) Parse() (net.IP, *net.IPNet, error) {
	return net.ParseCIDR(c.String())
}

// IsValid returns true of the cidr range is valid
func (c IPv4CIDR) IsValid() bool {
	_, _, err := c.Parse()
	return err == nil
}

// AddressCount returns the number of distinct host addresses within the given
// CIDR block.
//
// An error is returned if any of the CIDR blocks are invalid.
func (c IPv4CIDR) AddressCount() (uint64, error) {
	_, net, err := c.Parse()
	if err != nil {
		return 0, fmt.Errorf("failed to parse CIDR block %q: %w", c, err)
	}
	return cidr.AddressCount(net), nil
}
