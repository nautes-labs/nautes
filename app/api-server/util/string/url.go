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
	"fmt"
	"net"
	"net/url"
	"regexp"
)

type URL struct {
	HostName string
	Port     string
}

func GetURl(address string) (*URL, error) {
	ok := CheckURL(address)
	if !ok {
		return nil, fmt.Errorf("the address %s is not valid", address)
	}

	hostName, err := ParseUrl(address)
	if err != nil {
		return nil, err
	}

	port, err := ExtractPortFromURL(address)
	if err != nil {
		return nil, err
	}

	return &URL{
		HostName: hostName,
		Port:     port,
	}, nil
}

func ExtractPortFromURL(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	if u.Hostname() == "" {
		return "", nil
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return port, nil
}

func ParseUrl(urlString string) (string, error) {
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	// Check if the protocol is HTTP or HTTPS
	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("the URL is not using the HTTP or HTTPS protocol")
	}

	// Parse the host part to get the IP address or hostname
	host, _, err := net.SplitHostPort(parsedUrl.Host)
	if err != nil {
		// If the host part doesn't contain a port number, SplitHostPort returns an error.
		// In this case, we can assume that the host part is a hostname or an IP address.
		host = parsedUrl.Host
	}

	// Try to resolve the hostname to an IP address
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("the host part of the URL is not a valid hostname or IP address")
	}

	// Use the first resolved IP address
	return ips[0].String(), nil
}

func GetDomain(u string) string {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return ""
	}

	return parsedURL.Hostname()
}

func IsIp(u string) (bool, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false, err
	}

	host := parsedURL.Hostname()

	if ip := net.ParseIP(host); ip != nil {
		return true, nil
	}

	return false, nil
}

func IsIPPortURL(urlString string) bool {
	u, err := url.Parse(urlString)
	if err != nil {
		return false
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)

	return ip != nil
}

func CheckURL(address string) bool {
	regex := regexp.MustCompile(`^(https?://)`)
	return regex.MatchString(address)
}
