package argocd

import (
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	"k8s.io/apimachinery/pkg/util/sets"
)

type applications []syncer.Application

func (apps applications) GetCodeRepoUsage() CodeReposUsage {
	usage := CodeReposUsage{}
	for _, app := range apps {
		if app.Git != nil && app.Git.CodeRepo != "" {
			usage.AddCodeRepoUsage(app.Git.CodeRepo, []string{app.Name})
		}
	}
	return usage
}

func (apps applications) GetApps() []string {
	appSet := sets.New[string]()
	for _, app := range apps {
		index := appIndex{
			Product: app.Product,
			Name:    app.Name,
		}
		appSet.Insert(index.GetIndex())
	}
	return appSet.UnsortedList()
}
