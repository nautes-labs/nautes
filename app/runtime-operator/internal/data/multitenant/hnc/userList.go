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
