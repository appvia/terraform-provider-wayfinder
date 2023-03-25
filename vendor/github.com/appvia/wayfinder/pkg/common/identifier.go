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

package common

import (
	"sync"
)

var (
	idLock             = &sync.RWMutex{}
	instanceIdentifier string
)

// SetInstanceIdentifier sets the current instance identifier for this instance of Wayfinder
func SetInstanceIdentifier(ident string) {
	idLock.Lock()
	defer idLock.Unlock()

	instanceIdentifier = ident
}

// InstanceIdentifier returns the current instance identifier for this instance of Wayfinder, if
// known. This might be a blank string early in initialisation. Oh well.
func InstanceIdentifier() string {
	idLock.RLock()
	defer idLock.RUnlock()

	return instanceIdentifier
}

// ShortenInstanceID shortens the provided instance ID to a max of 6 characters for annoying clouds
// like GCP who have really tight name length limits. Should be reasonably unique, but can't
// guarantee globally unique.
func ShortenInstanceID(instanceID string) string {
	instID := instanceID
	if len(instID) > 6 {
		instID = instID[len(instID)-6:]
	}
	return instID
}

// InstanceIdentifierShort returns the last few characters of the instance ID - should be enough to
// be reasonably unique.
func InstanceIdentifierShort() string {
	return ShortenInstanceID(InstanceIdentifier())
}
