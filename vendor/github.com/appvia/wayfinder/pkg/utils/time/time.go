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

package time

import (
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/duration"
)

// HumanDuration returns a more human friendly duration
func HumanDuration(d time.Duration) string {
	return duration.HumanDuration(d)
}

// ParseUnixTimestamp attempts to convert the input returning the time and
// if valid boolean
func ParseUnixTimestamp(v string) (time.Time, bool) {
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return time.Unix(i, 0), true
}
