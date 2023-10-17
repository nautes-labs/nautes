// Copyright 2023 Nautes Authors
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

package loadcert

import (
	"crypto/x509"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	NautesDefaultCertsPath = "/opt/nautes/cert"
)

type LoadOptions struct {
	path string
}

type LoadOption func(*LoadOptions)

func CertPath(path string) LoadOption {
	return func(opts *LoadOptions) { opts.path = path }
}

func GetCertPool(opts ...LoadOption) (*x509.CertPool, error) {
	loadOpts := &LoadOptions{
		path: NautesDefaultCertsPath,
	}

	for _, fn := range opts {
		fn(loadOpts)
	}

	caCertPool := x509.NewCertPool()
	files, err := os.ReadDir(loadOpts.path)
	if err != nil {
		return nil, fmt.Errorf("list certs in %s failed: %w", loadOpts.path, err)
	}

	for _, file := range files {
		fullPath := filepath.Join(loadOpts.path, file.Name())

		ok, err := checkFileIsReadable(loadOpts.path, file)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		caBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read cert file %s failed: %w", file.Name(), err)
		}

		caCertPool.AppendCertsFromPEM(caBytes)
	}
	return caCertPool, nil
}

func checkFileIsReadable(rootPath string, file fs.DirEntry) (bool, error) {
	fullPath := filepath.Join(rootPath, file.Name())
	if file.IsDir() {
		return false, nil
	}

	info, err := file.Info()
	if err != nil {
		return false, fmt.Errorf("get file %s info failed: %w", file.Name(), err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		isDir, err := linkIsDir(rootPath, fullPath)
		if err != nil {
			return false, err
		}
		if isDir {
			return false, nil
		}
	}

	return true, nil
}

func linkIsDir(root, path string) (bool, error) {
	linkFile, err := os.Readlink(path)
	if err != nil {
		return false, fmt.Errorf("read link file %s failed: %w", path, err)
	}

	fullPath := linkFile
	if !strings.HasPrefix(linkFile, "/") {
		fullPath = filepath.Join(root, linkFile)
	}

	linkInfo, err := os.Lstat(fullPath)
	if err != nil {
		return false, fmt.Errorf("load link file %s failed: %w", path, err)
	}
	return linkInfo.IsDir(), nil
}
