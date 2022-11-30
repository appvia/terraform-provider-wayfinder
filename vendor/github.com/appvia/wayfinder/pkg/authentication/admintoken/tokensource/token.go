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

package tokensource

import (
	"fmt"
	"sync"
)

var (
	currToken = ""
	tokenLock = &sync.RWMutex{}
)

// SetToken sets the current runtime admin token
func SetToken(token string) error {
	tokenLock.Lock()
	defer tokenLock.Unlock()

	if l := len(token); l > 0 && l < 32 {
		currToken = ""
		return fmt.Errorf("admin token must be minimum 32 characters, admin token disabled")
	}

	currToken = token

	return nil
}

// GetToken returns the current token
func GetToken() string {
	tokenLock.RLock()
	defer tokenLock.RUnlock()

	return currToken
}

// Enabled returns true if an admin token exists
func Enabled() bool {
	return currToken != ""
}
