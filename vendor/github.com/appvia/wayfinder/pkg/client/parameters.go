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

package client

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PathParameters creates and returns a path param
func PathParameter(path, value string) ParameterFunc {
	return func() (Parameter, error) {
		if path == "" {
			panic("path parameter name cannot be empty")
		}
		if value == "" {
			panic(fmt.Errorf("%q path parameter cannot be empty", path))
		}

		return Parameter{
			IsPath: true,
			Name:   path,
			Value:  value,
		}, nil
	}
}

// QueryParameters creates and returns mutltiple funs
func QueryParameters(name string, values []string) []ParameterFunc {

	var list []ParameterFunc

	for _, x := range values {
		list = append(list, QueryParameter(name, x))
	}

	return list
}

// LabelParameter adds a label parameter to the query
func LabelParameter(name, value string) ParameterFunc {
	return func() (Parameter, error) {
		if name == "" {
			panic("name parameter not be empty")
		}
		if value == "" {
			panic("value parameter not be empty")
		}

		return Parameter{
			Name:  "label",
			Value: fmt.Sprintf("label=%s=%s", name, value),
		}, nil
	}
}

// QueryParameter creates and returns a query param
func QueryParameter(name, value string) ParameterFunc {
	return func() (Parameter, error) {
		if name == "" {
			return Parameter{}, fmt.Errorf("%s query parameter not be empty", name)
		}

		return Parameter{
			Name:  name,
			Value: value,
		}, nil
	}
}

// ForceParameter requests ignoring the read-only / ownership annotations/labels
func ForceParameter() ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "force",
			Value: "true",
		}, nil
	}
}

// OwnerParameter runs the operation with the specified owner
func OwnerParameter(owner string) ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "owner",
			Value: owner,
		}, nil
	}
}

// DryRunParameter requests a server-side dry run of the operation
func DryRunParameter() ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "dryRun",
			Value: metav1.DryRunAll,
		}, nil
	}
}

// ApplyParameter requests a server-side apply for an update operation
func ApplyParameter() ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "apply",
			Value: "true",
		}, nil
	}
}

// OrphanParameter requests deletion with orphaning (i.e. not deleting underlying cloud resources)
func OrphanParameter() ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "orphan",
			Value: "true",
		}, nil
	}
}

func CascadeParameter() ParameterFunc {
	return func() (Parameter, error) {
		return Parameter{
			Name:  "cascade",
			Value: "true",
		}, nil
	}
}
