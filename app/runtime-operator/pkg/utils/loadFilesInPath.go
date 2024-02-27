// Copyright 2024 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("utils")

// LoadFunc is the function that is called for each file.
type LoadFunc func(data []byte) error

// LoadFilesInPath loads all files in the given path and calls the given function for each file.
func LoadFilesInPath(rootPath string, fn LoadFunc) error {
	files, err := os.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("failed to read dir: %w", err)
	}

	for _, file := range files {
		var err error
		var ruleByte []byte

		info, err := file.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		path := filepath.Join(rootPath, file.Name())
		// Check if the file is a soft link
		if info.Mode()&os.ModeSymlink != 0 {

			// The file is a soft link
			// Read the soft link
			path, err = os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read soft link: %w", err)
			}

			path, err = GetAbsolutePath(rootPath, path)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}
		}

		// Check if the target is a directory
		targetInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to get target info: %w", err)
		}
		if targetInfo.IsDir() {
			continue
		}

		ruleByte, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Call the load function
		if err = fn(ruleByte); err != nil {
			return fmt.Errorf("failed to load file: %w", err)
		}
	}

	return nil
}

func GetAbsolutePath(rootPath, path string) (string, error) {
	// Check if the target is a directory
	if filepath.IsAbs(path) {
		return path, nil
	}

	if filepath.IsAbs(rootPath) {
		var err error
		rootPath, err = filepath.Abs(rootPath)
		if err != nil {
			return "", err
		}
	}

	targetPath := filepath.Join(rootPath, path)

	return targetPath, nil
}
