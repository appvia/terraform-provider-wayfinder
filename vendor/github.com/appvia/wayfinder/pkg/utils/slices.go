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
	"fmt"
	"sort"
	"strings"
)

// GetItemIfExists returns an item from a slice if there
func GetItemIfExists(list []string, index int) string {
	if index < 0 {
		return ""
	}
	if index < len(list) {
		return list[index]
	}

	return ""
}

// ContainsMatchingItems checks at least one item matches in the lists
func ContainsMatchingItems(a, b []string) bool {
	for _, x := range a {
		if Contains(x, b) {
			return true
		}
	}

	return false
}

// DeleteFromSlice removes a value from the slice
func DeleteFromSlice(v string, list []string) []string {
	var nl []string

	for _, x := range list {
		if v == x {
			continue
		}

		nl = append(nl, x)
	}

	return nl
}

// ToLower returns the strins in lower case
func ToLower(v []string) []string {
	list := make([]string, len(v))

	for i := 0; i < len(v); i++ {
		list[i] = strings.ToLower(v[i])
	}

	return list
}

// Contains checks a list has a value in it
func Contains(v string, l []string) bool {
	for _, x := range l {
		if v == x {
			return true
		}
	}

	return false
}

// ChunkBy breaks a slice into chunks
func ChunkBy(items []string, chunkSize int) (chunks [][]string) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}

// Unique removes any duplicates from a slice
func Unique(items []string) []string {
	var list []string

	found := make(map[string]bool)

	for _, x := range items {
		if ok := found[x]; !ok {
			list = append(list, x)

			found[x] = true
		}
	}

	return list
}

// StringsSorted returns a 'copy' of a sorted list of strings
func StringsSorted(list []string) []string {
	v := make([]string, len(list))
	copy(v, list)

	sort.Strings(v)

	return v
}

func StringSliceFrom(v interface{}) ([]string, bool) {
	if v == nil {
		return nil, true
	}

	switch vt := v.(type) {
	case []string:
		return vt, true
	case []interface{}:
		var res []string
		for _, e := range vt {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			res = append(res, s)
		}
		return res, true
	default:
		return nil, false
	}
}

type StringSet []string

func (s *StringSet) Add(v string) {
	for _, e := range *s {
		if e == v {
			return
		}
	}
	*s = append(*s, v)
}

func (s *StringSet) Remove(v string) {
	for i, e := range *s {
		if e == v {
			*s = append((*s)[0:i], (*s)[i+1:]...)
		}
	}
}

func (s *StringSet) MemberIf(v string, cond bool) {
	if cond {
		s.Add(v)
	} else {
		s.Remove(v)
	}
}

func (s *StringSet) Contains(v string) bool {
	for _, e := range *s {
		if e == v {
			return true
		}
	}
	return false
}

func StringSliceEquals(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	sort.Strings(s1)
	sort.Strings(s2)

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

// SliceToMap converts a slice of key=value,... to a map
func SliceToMap(list []string) (map[string]interface{}, error) {

	values := make(map[string]interface{})

	for _, item := range list {
		e := strings.Split(item, "=")
		if len(e) != 2 {
			return nil, fmt.Errorf("invalid param: %s", item)
		}
		switch strings.Contains(e[1], ",") {
		case true:
			values[e[0]] = strings.Split(e[1], ",")

		default:
			values[e[0]] = e[1]
		}
	}

	return values, nil
}
