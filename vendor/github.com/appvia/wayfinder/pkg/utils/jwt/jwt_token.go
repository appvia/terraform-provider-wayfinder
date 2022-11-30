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

package jwt

// JWTToken includes the raw token and claims
type JWTToken struct {
	*Claims
	// RawToken is the raw token
	RawToken []byte
}

// NewJWTTokenFromBytes creates and returns the parsed token
func NewJWTTokenFromBytes(token []byte) (*JWTToken, error) {
	claims, err := NewClaimsFromRawBytesToken(token)
	if err != nil {
		return nil, err
	}

	return &JWTToken{Claims: claims, RawToken: token}, nil
}
