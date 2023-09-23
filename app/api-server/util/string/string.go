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

package string

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}

	return false
}

func AddIfNotExists(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

func RemoveStringFromSlice(slice []string, value string) []string {
	for i := 0; i < len(slice); i++ {
		if slice[i] == value {
			slice[i] = slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			i--
		}
	}
	return slice
}

func ExtractNumber(prefix, str string) (int, error) {
	if !strings.HasPrefix(str, prefix) {
		return 0, fmt.Errorf("string %s does not have the expected prefix %s", str, prefix)
	}
	numStr := strings.TrimPrefix(str, prefix)
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func ReplaceSeparator(s, oldSep, newSep string) string {
	return strings.ReplaceAll(s, oldSep, newSep)
}

func ParseMetadataString(metadataStr string) map[string]string {
	metadataMap := make(map[string]string)
	splits := strings.Split(metadataStr, ",")
	for _, split := range splits {
		keyValue := strings.Split(split, "=")
		if len(keyValue) == 2 {
			metadataMap[keyValue[0]] = keyValue[1]
		}
	}
	return metadataMap
}

func EncodeToString(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func FirstCharToLower(str string) string {
	firstChar := string(str[0])
	lowerFirstChar := strings.ToLower(firstChar)
	result := lowerFirstChar + str[1:]

	return result
}

func ReplacePath(filePath, old, new string) (newPath string) {
	return strings.Replace(filePath, old, new, 1)
}

func IsBase64Decoding(str string) bool {
	if len(str)%4 != 0 {
		return false
	}

	for _, ch := range str {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=') {
			return false
		}
	}

	return true
}
