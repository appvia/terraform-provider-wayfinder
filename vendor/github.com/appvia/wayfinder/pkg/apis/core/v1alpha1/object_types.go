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
	"k8s.io/apimachinery/pkg/runtime"
)

// Object is the standard interface implemented by Wayfinder CRDs
// +kubebuilder:object:generate=false
type Object interface {
	// APIPath returns the path to find this object on the Wayfinder API (typically the lower-case
	// plural version of the kind). If the object is not served by the Wayfinder API, this should
	// return an empty string.
	APIPath() string
	CommonStatusAware
	runtime.Object
	metav1.Object
	// Clone returns a copy of this object as an object
	Clone() Object
	// CloneInto copies this object into the provided object
	CloneInto(Object)
	// ListType returns an empty example of the list type for this object
	ListType() ObjectList
}

// ObjectList is the standard interface implemented by Wayfinder list CRDs
// +kubebuilder:object:generate=false
type ObjectList interface {
	metav1.ListInterface
	runtime.Object
	// Clone returns a copy of this list as an object list
	Clone() ObjectList
	// CloneInto copies this list into the provided list
	CloneInto(ObjectList)
	// ObjectType returns an empty example object of the object type this list contains
	ObjectType() Object
	// GetItems returns the current list of items in this list
	GetItems() []Object
	// SetItems sets the current list of items in this list
	SetItems([]Object)
}
