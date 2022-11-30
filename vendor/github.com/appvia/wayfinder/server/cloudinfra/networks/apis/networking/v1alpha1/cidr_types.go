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
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

// IPv4CIDRBlocks is a list of IPv4 addresses
type IPv4CIDRBlocks []IPv4CIDR

// Strings returns the cidr blocks a string slice
func (c IPv4CIDRBlocks) Strings() []string {
	cidrBlocks := make([]string, len(c))
	for i, cidrBlock := range c {
		cidrBlocks[i] = cidrBlock.String()
	}
	return cidrBlocks
}

// IPv4CIDR represents an IP range in the A.B.C.D/N format
// +kubebuilder:validation:Pattern=`^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}/([0-9]|[1-2][0-9]|3[0-2])$`
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

// CheckOverlap takes a list subnets and and verifies none of them overlap, and
// all of them are within this CIDR block.
//
// An error is returned if any of the CIDR blocks are invalid, or if any of the
// above conditions are not satisfied.
func (c IPv4CIDR) CheckOverlap(cidrs []IPv4CIDR) error {
	_, supernet, err := c.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse supernet CIDR block %q: %w", supernet, err)
	}

	subnets := make([]*net.IPNet, len(cidrs))
	for i, s := range cidrs {
		_, subnet, err := s.Parse()
		if err != nil {
			return fmt.Errorf("failed to parse subnet CIDR block %q: %w", supernet, err)
		}
		subnets[i] = subnet
	}

	return cidr.VerifyNoOverlap(subnets, supernet)
}
