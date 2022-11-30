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

package strings

import (
	"fmt"
	"regexp"
	"strings"
)

// TruncateString truncates a string `s` to length `n`, replacing the removed characters with an
// ellipsis.
func TruncateString(s string, n int) string {
	if len(s) > n {
		ellipsis := "..."
		if n > len(ellipsis) {
			n -= len(ellipsis)
		}
		s = strings.TrimRight(s[:n], " .")
		s += ellipsis
	}
	return s
}

// StringScanRegex does sscanf stuff but with regex
func StringScanRegex(input string, expression string, output ...*string) error {
	re, err := regexp.Compile(expression)
	if err != nil {
		return err
	}
	result := re.FindStringSubmatch(input)
	if result == nil {
		return fmt.Errorf("no matches found")
	}
	if len(result) != len(output)+1 {
		return fmt.Errorf("%d output variables provided for %d submatches", len(output), len(result)-1)
	}
	for i, s := range output {
		*s = (result[i+1])
	}
	return nil
}
