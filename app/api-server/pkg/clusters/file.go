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

package cluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	ClusterTemplatesDir = "cluster-tempaltes"
)

type FileOperation interface {
	ReadFile(filePath string) ([]byte, error)
	CreateFile(filePath string) (string, error)
	WriteFile(filePath string, content []byte) error
	DeleteFile(filePath string) error
	CreateDir(dir string) (string, error)
	DeleteDir(dir string) error
	IsDir(path string) (bool, error)
	ListFilesInDirectory(dirPath string) ([]string, error)
}

type File struct{}

func NewClusterConfigFile() (FileOperation, error) {
	return &File{}, nil
}

func (c *File) CreateFile(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, os.ModeAppend)
	if err != nil {
		return "", err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}

func (c *File) ReadFile(filePath string) ([]byte, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *File) DeleteFile(filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	return os.Remove(filePath)
}

func (c *File) WriteFile(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	err := ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *File) CreateDir(dir string) (string, error) {
	dir = fmt.Sprintf("%s-%d", dir, time.Now().Unix())

	_, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		return dir, nil
	}

	if err = os.MkdirAll(dir, os.ModeAppend); err != nil {
		return "", err
	}

	return dir, nil
}

func (c *File) IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return fileInfo.IsDir(), nil
}

func (c *File) ListFilesInDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (c *File) DeleteDir(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	return os.RemoveAll(dir)
}
