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

// Components is a collection child components for a resource
type Components []*Component

// Component the state of a component of the resource
type Component struct {
	// Name is the name of the component
	Name string `json:"name,omitempty"`
	// Status is the status of the component
	Status Status `json:"status,omitempty"`
	// Message is a human readable message on the status of the component
	Message string `json:"message,omitempty"`
	// Detail is additional details on the error is any
	Detail string `json:"detail,omitempty"`
	// Resource is a reference to the resource
	Resource *Ownership `json:"resource,omitempty"`
}

// SetComponent sets the state of a component
func (c *Components) SetComponent(component Component) {
	item, found := c.GetComponent(component.Name)
	if !found {
		*c = append(*c, &component)

		return
	}
	item.Status = component.Status
	item.Message = component.Message
	item.Detail = component.Detail
}

// GetComponent checks if the status of a component exists
func (c Components) GetComponent(name string) (*Component, bool) {
	for _, x := range c {
		if x.Name == name {
			return x, true
		}
	}

	return nil, false
}

func (c *Components) RemoveComponent(name string) {
	for i, x := range *c {
		if x.Name == name {
			*c = append((*c)[:i], (*c)[i+1:]...)
		}
	}
}
