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
	"encoding/json"
	"errors"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
)

var (
	// ErrMapLookupNotFound indicates the value was not found
	ErrMapLookupNotFound = errors.New("value not found")
)

// ContainsMap checks if the secondary maps contains all the values from the
// first argument
func ContainsMap(expected, have map[string]string) bool {
	switch {
	case have == nil:
		return false
	case expected == nil:
		return true
	}

	for k, v := range expected {
		if have[k] != v {
			return false
		}
	}

	return true
}

// MergeStringMaps merges maps together
func MergeStringMaps(a, b map[string]string) map[string]string {
	m := make(map[string]string)

	switch {
	case a == nil && b == nil:
		return m
	case a == nil && b != nil:
		return b
	case b == nil && a != nil:
		return a
	}

	for _, x := range []map[string]string{a, b} {
		for k, v := range x {
			m[k] = v
		}
	}

	return m
}

// CovertStingMapToInterface returns map of strings to map[string]interface
func CovertStingMapToInterface(a map[string]string) map[string]interface{} {
	m := make(map[string]interface{})
	if a == nil {
		return m
	}

	for k, v := range a {
		m[k] = v
	}

	return m
}

// MapKeys returns the keys of a map
func MapKeys(v map[string]string) []string {
	keys := make([]string, len(v))
	index := 0

	for k := range v {
		keys[index] = k
		index++
	}

	return keys
}

// MapLookup is capable of lookup up a value in a nested map
func MapLookup(m map[string]interface{}, ks ...string) (rval interface{}, err error) {
	var ok bool

	if len(ks) == 0 { // degenerate input
		return nil, fmt.Errorf("NestedMapLookup needs at least one key")
	}
	if rval, ok = m[ks[0]]; !ok {
		return nil, ErrMapLookupNotFound
	} else if len(ks) == 1 { // we've reached the final key
		return rval, nil
	} else if m, ok = rval.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("malformed structure at %#v", rval)
	} else { // 1+ more keys
		return MapLookup(m, ks[1:]...)
	}
}

func PatchMap(m, patch map[string]interface{}) (map[string]interface{}, error) {
	if m == nil {
		m = map[string]interface{}{}
	}

	jm, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	jp, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	jres, err := jsonpatch.MergePatch(jm, jp)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	if err := json.Unmarshal(jres, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func StringMapEquals(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k1, v1 := range m1 {
		v2, ok := m2[k1]
		if !ok || v2 != v1 {
			return false
		}
	}

	return true
}
