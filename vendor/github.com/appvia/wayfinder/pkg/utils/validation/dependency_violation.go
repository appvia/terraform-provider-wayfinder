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

package validation

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/appvia/wayfinder/pkg/common"
	stringutils "github.com/appvia/wayfinder/pkg/utils/strings"
)

const CauseTypeDependencyViolation metav1.CauseType = "DependencyViolation"

type ErrDependencyViolation struct {
	Message    string               `json:"message"`
	Dependents []DependentReference `json:"dependents"`
}

func (e ErrDependencyViolation) Error() string {
	var str string
	for _, d := range e.Dependents {
		if !d.System {
			str = fmt.Sprintf("%s * %s\n", str, d.String())
		}
	}

	if str != "" {
		if e.Message == "" {
			e.Message = "the following objects need to be deleted first"
		}
		return strings.TrimRight(e.Message, ":") + ":\n" + str
	}

	for _, d := range e.Dependents {
		str = str + fmt.Sprintf(" * %s\n", d.String())
	}

	return "waiting for the following objects to be deleted by Wayfinder:\n" + str
}

// DependentReference is an object reference to a dependent object in the same namespace
type DependentReference struct {
	// API version of the dependent
	APIVersion string `json:"apiVersion"`
	// Kind of the dependent
	Kind string `json:"kind"`
	// Name of the dependent
	Name string `json:"name"`
	// System is true if this is a system resource
	System bool `json:"system"`
}

func (d DependentReference) String() string {
	return fmt.Sprintf("%s/%s/%s", d.APIVersion, d.Kind, d.Name)
}

// DependentReferenceFromObject creates a DependentReference from the given object
func DependentReferenceFromObject(o client.Object) DependentReference {
	return DependentReference{
		APIVersion: o.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       o.GetObjectKind().GroupVersionKind().Kind,
		Name:       o.GetName(),
		System:     o.GetAnnotations()[common.AnnotationSystem] == common.AnnotationValueTrue,
	}
}

// DependentReferenceFromStrong converts the string representation of a DependentReference
// back into a DependentReference
func DependentReferenceFromString(input string) (DependentReference, error) {
	d := DependentReference{}
	err := stringutils.StringScanRegex(input, "^(.+/.+)/(.+)/(.+)$", &d.APIVersion, &d.Kind, &d.Name)
	return d, err
}
