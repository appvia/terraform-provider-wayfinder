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
