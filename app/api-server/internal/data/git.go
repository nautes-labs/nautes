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

package data

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

const (
	_DefaultProject = "/tmp/product"
)

type gitRepo struct {
	config *nautesconfigs.Config
}

func extractRepoName(repoURL string) (string, error) {
	re := regexp.MustCompile(`^.+/(.+)\.git$`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) != 2 {
		return "", fmt.Errorf("failed to extract repo name from URL %q", repoURL)
	}

	return strings.TrimSpace(matches[1]), nil
}

func (g *gitRepo) Clone(ctx context.Context, param *biz.CloneRepositoryParam) (string, error) {
	if param == nil {
		return "", fmt.Errorf("please check that the parameters, url, user and email are not allowed to be empty")
	}

	localRepositarySubPath := fmt.Sprintf("%s%v", _DefaultProject, time.Now().Unix())
	err := os.MkdirAll(localRepositarySubPath, os.FileMode(0777))
	if err != nil {
		return "", err
	}

	// clone product config repository according to token
	token, ok := ctx.Value("token").(string)
	if !ok {
		return "", fmt.Errorf("token must be string type")
	}
	url := strings.Replace(param.URL, "https://", "", 1)
	gitCloneURL := fmt.Sprintf("https://%s:%s@%s", param.User, token, url)
	cmd := exec.Command("git", "clone", "--verbose", gitCloneURL)
	cmd.Dir = localRepositarySubPath
	data, err := cmd.CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("failed to clone the repository: %s, err: %w", url, err)
	}

	repoName, err := extractRepoName(param.URL)
	if err != nil {
		return "", err
	}
	// set git config user name
	localRepositoryPath := fmt.Sprintf("%s/%s", localRepositarySubPath, repoName)
	intput1 := fmt.Sprintf("config user.name %s", param.User)
	cmd1 := exec.Command("git", intput1)
	cmd1.Dir = localRepositoryPath
	err = cmd1.Run()
	if err != nil {
		return "", fmt.Errorf("failed to set git user user in %s, err: %w", localRepositoryPath, err)
	}

	// set git config user email
	input2 := fmt.Sprintf("config user.email %s", param.Email)
	cmd2 := exec.Command("git", input2)
	cmd2.Dir = localRepositoryPath
	err = cmd2.Run()
	if err != nil {
		return "", fmt.Errorf("failed to set git user email in %s, err: %w", localRepositoryPath, err)
	}

	return localRepositoryPath, nil
}

func (g *gitRepo) Diff(_ context.Context, path string, command ...string) (string, error) {
	cmd := exec.Command("git", "diff")
	cmd.Args = append(cmd.Args, command...)
	cmd.Dir = path
	data, err := cmd.CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("diff data: %v, err: %w", string(data), err)
	}

	return string(data), nil
}

func (g *gitRepo) Fetch(_ context.Context, path string, command ...string) (string, error) {
	cmd := exec.Command("git", "fetch")
	cmd.Args = append(cmd.Args, command...)
	cmd.Dir = path
	data, err := cmd.CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("fetch data: %v, err: %w", string(data), err)
	}

	return string(data), nil
}

func (g *gitRepo) Merge(_ context.Context, path string) (string, error) {
	cmd := exec.Command("git", "merge")
	cmd.Dir = path
	data, err := cmd.CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("merge data: %v, err: %w", string(data), err)
	}

	return string(data), nil
}

func (g *gitRepo) Status(path string) (string, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", err
	}

	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	status, err := w.Status()
	if err != nil {
		return "", err
	}

	return status.String(), nil
}

func (g *gitRepo) Commit(ctx context.Context, path string) error {
	cmdAdd := exec.Command("git", "add", ".")
	cmdAdd.Dir = path
	_, err := cmdAdd.CombinedOutput()
	if err != nil {
		return err
	}

	commit := generateCommitMessage(ctx)
	cmdCommit := exec.Command("git", "commit", "-m", commit)
	cmdCommit.Dir = path
	data, err := cmdCommit.CombinedOutput()
	if err != nil {
		return fmt.Errorf("commit output: %v, err: %w", string(data), err)
	}

	return nil
}

// generateCommitMessage is commit message through resource information splicing.
// eg: [API] Save_CodeRepoBinding: coderepobinding1, Product: test-product, CodeRepo: repo-1.
func generateCommitMessage(ctx context.Context) string {
	var commitMsg string

	value := ctx.Value(biz.ResourceInfoKey)
	info, ok := value.(*biz.ResourceInfo)
	if !ok {
		return "[API] Save operator"
	}
	// If request resource kind is product,
	// It means that when creating a product, the default commit message is initial.
	if info.ResourceKind == nodestree.Product && info.Method == biz.SaveMethod {
		commitMsg = "initial commit."
		return commitMsg
	}

	commitMsg = fmt.Sprintf("[API] %s_%s: %s", info.Method, info.ResourceKind, info.ResourceName)

	// When creating a cluster, the product name may be empty.
	if info.ProductName != "" {
		commitMsg += fmt.Sprintf(", Product: %s", info.ProductName)
	}

	// If the parent resource is not empty, increase the information of the parent resource.
	if info.ParentResouceKind != "" && info.ParentResourceName != "" {
		commitMsg += fmt.Sprintf(", %s: %s.", info.ParentResouceKind, info.ParentResourceName)
	} else {
		commitMsg += "."
	}

	return commitMsg
}

func (g *gitRepo) Push(_ context.Context, path string, command ...string) error {
	cmd := exec.Command("git", "push")
	cmd.Args = append(cmd.Args, command...)
	cmd.Dir = path
	data, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("push data: %v, err: %w", string(data), err)
	}

	return nil
}

func (g *gitRepo) SaveConfig(ctx context.Context, path string) error {
	status, err := g.Status(path)
	if err != nil {
		return err
	}

	if status != "" {
		err = g.Commit(ctx, path)
		if err != nil {
			return err
		}

		err = g.Push(ctx, path)
		if err != nil {
			return err
		}
	}

	return nil
}
