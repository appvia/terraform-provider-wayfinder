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

package client

import "github.com/appvia/wayfinder/pkg/client/config"

// OptionFunc is a option function
type OptionFunc func(*cc)

// UseUpdateHandler sets the update handler
func UseUpdateHandler(handle UpdateHandlerFunc) OptionFunc {

	return func(c *cc) {
		c.handler = handle
	}
}

func UseAPIClient(clientSrc func(*config.Config) RestInterface) OptionFunc {
	return func(c *cc) {
		c.apiClient = clientSrc
	}
}
