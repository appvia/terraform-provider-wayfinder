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

import "regexp"

const (
	// AccountLocal is a local basic auth account
	AccountLocal = "basicauth"
	// AccountToken is a api token account
	AccountToken = "token"
	// AccountSSO is a openid account
	AccountSSO = "sso"
	// AdminUser is the default wayfinder admin user
	AdminUser = "admin"
)

var (
	// SupportedAccounts is a list of supported account types
	SupportedAccounts = []string{
		AccountLocal,
		AccountToken,
		AccountSSO,
	}
	// UsernameRegex is a filter for the username
	UsernameRegex = regexp.MustCompile(`^[a-z-A-Z0-9\@\-\._\+]{3,63}$`)
)
