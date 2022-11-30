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

package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// ErrMissingObject indicates no object defined
	ErrMissingObject = errors.New("missing object")
	// ErrNotRuntimeObject indicates the object is not a runtime.Object
	ErrNotRuntimeObject = errors.New("object is not a runtime.Object")
	// ErrNotMetaObject indicates the object does not implement metav1.Object
	ErrNotMetaObject = errors.New("object does not implement metav1.Object")
)

// ConvertToMap converts a struct to a map - note the fields must be
// exported for refection to work
func ConvertToMap(v interface{}) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	if v == nil {
		return values, nil
	}

	encoded := &bytes.Buffer{}
	if err := json.NewEncoder(encoded).Encode(v); err != nil {
		return nil, err
	}

	return values, json.NewDecoder(encoded).Decode(&values)
}

// GetObjectUID returns the object uuid
func GetObjectUID(o metav1.Object) string {
	if o == nil {
		return ""
	}

	return string(o.GetUID())
}

// GetMetaObject returns the metav1.Object interface
func GetMetaObject(obj interface{}) (metav1.Object, error) {
	// @step: retrieve the payload from the request
	if obj == nil {
		return nil, ErrMissingObject
	}
	mo, ok := obj.(metav1.Object)
	if !ok {
		return nil, ErrNotMetaObject
	}

	return mo, nil
}

// GetRuntimeObject returns the runtime.Object
func GetRuntimeObject(o interface{}) (runtime.Object, error) {
	if o == nil {
		return nil, ErrMissingObject
	}

	mo, ok := o.(runtime.Object)
	if !ok {
		return nil, ErrNotRuntimeObject
	}

	return mo, nil
}

// IsEmpty checks if a struct has any values set
func IsEmpty(v interface{}) bool {
	t := reflect.ValueOf(v).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsValid() || !field.IsZero() {
			return false
		}
	}

	return true
}
