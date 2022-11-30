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

package help

import (
	"embed"
	"fmt"
)

//go:embed *.md
var helpFS embed.FS

// HelpAsset will get a policy file asset or an error if it doesn't exist
func HelpAsset(name string) ([]byte, error) {
	return helpFS.ReadFile(name + ".md")
}

// MustHelpAsset will return a single file name asset or panic
func MustHelpAsset(name string) []byte {
	content, err := HelpAsset(name)
	if err != nil {
		panic(fmt.Errorf("embedded asset does not exist %s - %w", name, err))
	}
	return content
}
