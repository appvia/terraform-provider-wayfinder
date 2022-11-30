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

package file

import (
	"io"
	"os"
	"path"
	"path/filepath"
)

// HasFileExts checks the extension against a collection
func HasFileExts(file string, exts []string) bool {
	for _, x := range exts {
		if HasFileExt(file, x) {
			return true
		}
	}

	return false
}

// HasFileExt checks the file extension
func HasFileExt(file, ext string) bool {
	return path.Ext(file) == ext
}

// ExpandFilePaths expand x number of paths
func ExpandFilePaths(paths []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		switch path {
		case "-":
			files = append(files, path)

		default:
			list, err := ExpandFilePath(path)
			if err != nil {
				return nil, err
			}
			files = append(files, list...)
		}
	}

	return files, nil
}

// ExpandFilePath is used to pull in all the files
func ExpandFilePath(path string) ([]string, error) {

	var files []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// ReadFileOrStdin is responsible for reading from the file or stdin
func ReadFileOrStdin(stdin io.Reader, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}

	return os.ReadFile(path)
}

// DirExists checks if the directory exists
func DirExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return stat.IsDir(), nil
}

// FileExists checks if a file exists
func FileExists(filename string) (bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return !info.IsDir(), nil
}

// EnsureFileExists creates an (empty) file if filename does not exist (returning true)
// or does nothing if it already exists (returning false). Will create missing directories.
func EnsureFileExists(filename string) (bool, error) {
	exists, err := FileExists(filename)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return false, err
	}
	if _, err := os.Create(filename); err != nil {
		return false, err
	}

	return true, nil
}
