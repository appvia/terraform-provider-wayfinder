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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DurationPtr returns a pointer to a duration
func DurationPtr(value time.Duration) *time.Duration {
	return &value
}

// DurationValue returns the value of the duration
func DurationValue(value *time.Duration) time.Duration {
	if value == nil {
		return time.Duration(0)
	}

	return *value
}

// StringPtr returns a pointer to a string
func StringPtr(value string) *string {
	return &value
}

// TruePtr return a pointer to a true boolean
func TruePtr() *bool {
	yes := true

	return &yes
}

// BoolPtr returns the a pointer to the boolean
func BoolPtr(v bool) *bool {
	return &v
}

// IntValue returns the int pointer value
func IntValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// IntPtr returns a pointer to an int
func IntPtr(v int) *int {
	return &v
}

// Int64Ptr returns a pointer to value
func Int64Ptr(v int64) *int64 {
	return &v
}

// Int32Ptr returns a pointer to value
func Int32Ptr(v int32) *int32 {
	return &v
}

// Int64Value returns a pointer to value
func Int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}

	return *v
}

// MetaV1TimePtr returns a metav1.Time pointer
func MetaV1TimePtr(tm time.Time) *metav1.Time {
	t := metav1.NewTime(tm)

	return &t
}

// StringValue return the string or empty
func StringValue(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

// StringSlicePtr returns the string slive pointer
func StringSlicePtr(v []string) *[]string {
	return &v
}

// StringSliceValue returns the string slice
func StringSliceValue(v *[]string) []string {
	if v == nil {
		return []string{}
	}

	return *v
}

// BoolValue return the boolean value or false
func BoolValue(v *bool) bool {
	if v == nil {
		return false
	}

	return *v
}
