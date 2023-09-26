package argocd

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

func (u *userList) getUsers() []string {
	return u.Users.UnsortedList()
}

func (u *userList) getUsersAsString() string {
	return strings.Join(u.Users.UnsortedList(), ",")
}

func (u *userList) addUsers(users []string) {
	u.Users = sets.New(users...).Union(u.Users)
}

func (u *userList) deleteUsers(users []string) {
	u.Users = u.Users.Delete(users...)
}
