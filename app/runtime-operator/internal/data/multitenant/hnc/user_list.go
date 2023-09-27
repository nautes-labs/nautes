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

package hnc

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type userList struct {
	Users sets.Set[string]
}

func newUserList(userString string) userList {
	users := strings.Split(userString, ",")
	if userString == "" {
		users = []string{}
	}
	return userList{
		Users: sets.New(users...),
	}
}

func (u *userList) hasUser(name string) bool {
	return u.Users.Has(name)
}

func (u *userList) getUsers() []string {
	return u.Users.UnsortedList()
}

func (u *userList) getUsersAsString() string {
	return strings.Join(u.Users.UnsortedList(), ",")
}

func (u *userList) addUser(name string) {
	u.Users.Insert(name)
}

func (u *userList) deleteUser(name string) {
	u.Users.Delete(name)
}
