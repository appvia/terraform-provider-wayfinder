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
)

const zeroVersion = "0.0.0"

// FindLatest iterates over the provided versions and finds the latest.
func FindLatest(versions []string) (string, error) {
	return FindLatestWithPrefix(versions, "")
}

// FindLatestWithPrefix iterates over the provided versions and finds the latest
// version with the given prefix.
//
// A prefix can be a partial or full version, e.g. "1" / "1.2" / "1.2.3", and so on.
func FindLatestWithPrefix(versions []string, prefix string) (string, error) {
	if len(versions) == 0 {
		return "", nil
	}

	if prefix == "" {
		prefix = zeroVersion
	}

	p, err := NewConstraint("~" + prefix)
	if err != nil {
		return "", fmt.Errorf("failed to parse prefix %q: %w", prefix, err)
	}

	latest, err := NewVersion(zeroVersion)
	if err != nil {
		return "", fmt.Errorf("failed to initialize latest version %q: %w", latest, err)
	}

	found := false
	for _, version := range versions {
		v, err := NewVersion(version)
		if err != nil {
			return "", fmt.Errorf("failed to parse version %q: %w", version, err)
		}
		if p.Check(v) && v.GreaterThan(latest) {
			latest = v
			found = true
		}
	}
	if !found {
		return "", nil
	}

	return latest.Original(), nil
}
