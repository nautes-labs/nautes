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

package key

import (
	"fmt"
	"os"
	"os/exec"
)

func GenerateKeyPair(keyType, tag string) ([]byte, []byte, error) {
	keyPath := fmt.Sprintf("%s/%s", os.TempDir(), tag)
	cmd := exec.Command("ssh-keygen", "-t", keyType, "-P", "", "-f", keyPath, "-C", tag)
	err := cmd.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate secret key pair, err: %w", err)
	}

	defer os.Remove(keyPath)
	defer os.Remove(fmt.Sprintf("%s.pub", keyPath))

	publicKeyPath := fmt.Sprintf("%s.pub", keyPath)
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read public key file")
	}

	privateKeyPath := keyPath
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read public key file")
	}

	return publicKeyBytes, privateKeyBytes, nil
}
