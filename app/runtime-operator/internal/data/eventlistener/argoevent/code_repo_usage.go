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

package argoevent

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type codeRepoUsage struct {
	sets.Set[string]
}

func newCodeRepoUsage(userString string) codeRepoUsage {
	codeRepos := strings.Split(userString, ",")
	if userString == "" {
		codeRepos = []string{}
	}
	return codeRepoUsage{
		sets.New(codeRepos...),
	}
}

func (u *codeRepoUsage) ListAsString() string {
	return strings.Join(u.UnsortedList(), ",")
}
