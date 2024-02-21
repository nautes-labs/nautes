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

package utils

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/nautes-labs/nautes/pkg/nautesenv"
)

const certPath = "./cert"

func GetCABundle(serviceURL string) ([]byte, error) {
	u, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("parse url failed: %w", err)
	}

	var fileName string
	if u.Port() != "" {
		fileName = fmt.Sprintf("%s_%s.crt", u.Hostname(), u.Port())
	} else {
		fileName = fmt.Sprintf("%s.crt", u.Hostname())
	}

	fileFullPath := filepath.Join(nautesenv.GetNautesHome(), certPath, fileName)
	return os.ReadFile(fileFullPath)
}
