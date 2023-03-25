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

package semver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// GetOrderedPackageVersions takes in a valid slice of package versions and returns them in order
// N.B. Wayfinder package versions are constructed in the form `<chart-version>-<wayfinder-version>`
func GetOrderedPackageVersions(vers []string) ([]string, error) {
	vs := make([]*semver.Version, len(vers))
	for i, r := range vers {
		v, err := semver.NewVersion(r)
		if err != nil {
			return nil, fmt.Errorf("error parsing version: %w", err)
		}

		vs[i] = v
	}

	sort.Sort(semver.Collection(vs))

	res := []string{}
	for _, v := range vs {
		str := v.String()
		res = append(res, str)

	}

	return res, nil
}

// GetLatestPackageVersion takes in a valid slice of package versions and returns the latest semver
// N.B. Wayfinder package versions are constructed in the form `<chart-version>-<wayfinder-version>`
func GetLatestPackageVersion(vers []string) (string, error) {
	latest := vers[0]

	ver, err := FindLatest(vers)
	if err != nil {
		return "", fmt.Errorf("1failed to find latest version: %w", err)
	}

	if ver != "" {
		latest = ver
	}

	for _, v1 := range vers {
		for _, v2 := range vers {
			var internalLatest string
			if v1 != v2 && strings.Contains(v1, strings.Split(v2, "-")[0]) {
				v1inner := strings.Split(v1, "-")[1]
				v2inner := strings.Split(v2, "-")[1]

				ver, err := FindLatest([]string{v1inner, v2inner})
				if err != nil {
					return "", fmt.Errorf("2failed to find latest version: %w", err)
				}

				switch ver {
				case v1inner:
					internalLatest = v1
				case v2inner:
					internalLatest = v2
				}
			} else {
				v1inner := strings.Split(v1, "-")[0]
				v2inner := strings.Split(v2, "-")[0]

				ver, err := FindLatest([]string{v1inner, v2inner})
				if err != nil {
					return "", fmt.Errorf("3failed to find latest version: %w", err)
				}

				switch ver {
				case v1inner:
					internalLatest = v1
				case v2inner:
					internalLatest = v2
				}
			}

			if internalLatest != "" {
				if latest != internalLatest && strings.Contains(latest, strings.Split(internalLatest, "-")[0]) {
					vLatest := strings.Split(latest, "-")[1]
					vInternalLatest := strings.Split(internalLatest, "-")[1]

					ver, err := FindLatest([]string{vLatest, vInternalLatest})
					if err != nil {
						return "", fmt.Errorf("4failed to find latest version: %w", err)
					}

					switch ver {
					case vInternalLatest:
						latest = internalLatest
					}
				} else {
					vLatest := strings.Split(latest, "-")[0]
					vInternalLatest := strings.Split(internalLatest, "-")[0]

					ver, err := FindLatest([]string{vLatest, vInternalLatest})
					if err != nil {
						return "", fmt.Errorf("5failed to find latest version: %w", err)
					}

					switch ver {
					case vInternalLatest:
						latest = internalLatest
					}
				}
			}
		}
	}

	if latest == "" {
		err := fmt.Errorf("6failed to find latest version")
		return "", err
	}

	return latest, nil
}
