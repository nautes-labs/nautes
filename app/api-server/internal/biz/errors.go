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

package biz

import (
	"fmt"
	"strings"

	errors "github.com/go-kratos/kratos/v2/errors"
)

const (
	ProjectNotFoud   = "PROJECT_NOT_FOUND"
	GroupNotFound    = "GROUP_NOT_FOUND"
	NodeNotFound     = "NODE_NOT_FOUND"
	ResourceNotFound = "RESOURCE_NOT_FOUND"
	ResourceNotMatch = "RESOURCE_NOT_MATCH"
	NoAuthorization  = "NO_AUTHORIZATION"
	ResourceNotExist = "RESOURCE_NOT_EXIST"
)

var (
	ErrorProjectNotFound = errors.New(404, ProjectNotFoud, "the project path is not found")
	ErrorGroupNotFound   = errors.New(404, GroupNotFound, "the group path is not found")
	ErrorNodetNotFound   = errors.New(404, NodeNotFound, "the node is not found")
	ErrorResourceNoFound = errors.New(404, ResourceNotFound, "the resource is not found")
	ErrorResourceNoMatch = errors.New(500, ResourceNotMatch, "the resource is not match")
	ErrorNoAuth          = errors.New(403, NoAuthorization, "no access to the code repository")
)

const _ResourceDoesNotExistOrUnavailable = "during global validation, it was found that %s '%s' does not exist or is unavailable. Please check %s '%s' in directory '%s'"

func ResourceDoesNotExistOrUnavailable(resourceName, resourceKind, resourcePath string, errMsgs ...string) error {
	errMsg := fmt.Sprintf("During global validation, it was found that %s '%s' does not exist or is unavailable. Please check resource path of default project: %s", resourceName, resourceKind, resourcePath)

	if len(errMsgs) > 0 {
		errMsg = fmt.Sprintf("%s, err: %s", errMsg, strings.Join(errMsgs, "|"))
	}

	return errors.New(500, ResourceNotExist, errMsg)
}
