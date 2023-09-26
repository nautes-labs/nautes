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
