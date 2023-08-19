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

package error

import (
	"fmt"
	"strings"
)

func CollectAndCombineErrors(errChan chan interface{}) error {
	close(errChan)

	errMsgs := make([]string, 0)

	for err := range errChan {
		switch err := err.(type) {
		case string:
			errMsgs = append(errMsgs, err)
		case error:
			errMsgs = append(errMsgs, err.Error())
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("Unknown error type: %v", err))
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "|"))
	}

	return nil
}
