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
	"bytes"
	"crypto/rand"
	"math/big"
	"strings"
)

// GeneratePassword trys to generate a random password for us
func GeneratePassword(length int, allowSpecialChars bool) (string, error) {
	const (
		lowerCase    = "abcdefghijklmnopqrstvuwxyz"
		upperCase    = "ABCDEFGHIJKLMNOPQRSTVUWXYZ"
		numbers      = "0123456789"
		specialChars = "-=[];'\\`,./!@#$%^&*()_+{}:\"|~<>?"
	)

	chars := lowerCase + upperCase + numbers
	if allowSpecialChars {
		chars = chars + specialChars
	}

	var password string
	for {
		res := bytes.NewBuffer(make([]byte, 0, length))
		max := big.NewInt(int64(len(chars)))
		for i := 0; i < length; i++ {
			pos, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			res.WriteByte(chars[pos.Int64()])
		}
		password = res.String()

		if strings.ContainsAny(password, lowerCase) &&
			strings.ContainsAny(password, upperCase) &&
			strings.ContainsAny(password, numbers) &&
			(!allowSpecialChars || strings.ContainsAny(password, specialChars)) {
			break
		}
	}

	return password, nil
}
