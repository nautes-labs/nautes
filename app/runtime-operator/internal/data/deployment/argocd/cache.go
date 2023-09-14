package argocd

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type AppCache struct {
	AppNames       []string       `json:"appNames" yaml:"appNames"`
	CodeReposUsage CodeReposUsage `json:"codeReposUsage" yaml:"codeReposUsage"`
}

func newAppCache(cache interface{}) (*AppCache, error) {
	if cache == nil {
		return &AppCache{
			AppNames:       []string{},
			CodeReposUsage: map[string]CodeRepoUsage{},
		}, nil
	}

	oldCache, ok := cache.(*AppCache)
	if !ok {
		return nil, fmt.Errorf("cache is not a app cache")
	}

	newCache := *oldCache
	return &newCache, nil
}

func (ac *AppCache) AddApp(name string) {
	ac.AppNames = sets.New(ac.AppNames...).Insert(name).UnsortedList()
}

func (ac *AppCache) DeleteApp(name string) {
	ac.AppNames = sets.New(ac.AppNames...).Delete(name).UnsortedList()
}

type CodeReposUsage map[string]CodeRepoUsage

func (u CodeReposUsage) GetCodeRepoSet() sets.Set[string] {
	codeRepoSet := sets.New[string]()
	for _, usage := range u {
		codeRepoSet.Insert(usage.Name)
	}
	return codeRepoSet
}

func (u CodeReposUsage) AddCodeRepoUsage(repoName string, userNames []string) {
	usage, ok := u[repoName]
	if !ok {
		u[repoName] = CodeRepoUsage{
			Name:  repoName,
			Users: userNames,
		}
	} else {
		usage.AddUsers(userNames...)
		u[repoName] = usage
	}
}

func (u CodeReposUsage) DeleteCodeRepoUsage(repoName string, userNames []string) {
	usage, ok := u[repoName]
	if !ok {
		return
	}
	usage.DeleteUsers(userNames...)
	u[repoName] = usage

	if len(u[repoName].Users) == 0 {
		delete(u, repoName)
	}
}

type CodeRepoUsage struct {
	Name  string   `json:"name" yaml:"name"`
	Users []string `json:"Users" yaml:"Users"`
}

func (cru *CodeRepoUsage) AddUsers(users ...string) {
	cru.Users = sets.New(cru.Users...).Union(sets.New(users...)).UnsortedList()
}

func (cru *CodeRepoUsage) DeleteUsers(users ...string) {
	cru.Users = sets.New(cru.Users...).Delete(users...).UnsortedList()
}

func (cru CodeRepoUsage) getUsers() []string {
	return cru.Users
}

type appIndex struct {
	Product string
	Name    string
}

func newAppIndex(indexString string) appIndex {
	elements := strings.SplitN(indexString, "/", 2)
	return appIndex{
		Product: elements[0],
		Name:    elements[1],
	}
}

func (ai *appIndex) GetIndex() string {
	return fmt.Sprintf("%s/%s", ai.Product, ai.Name)
}
