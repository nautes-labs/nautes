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

package utils

import (
	"fmt"
	"net"
	"net/url"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type EntryPoint struct {
	Domain   string
	Port     int
	Protocol string
}

func (e *EntryPoint) GetURL() string {
	return fmt.Sprintf("%s://%s:%d", e.Protocol, e.Domain, e.Port)
}

func GetEntryPointFromCluster(cluster v1alpha1.Cluster, hostCluster *v1alpha1.Cluster, protocol string) (*EntryPoint, error) {
	entryPoint := EntryPoint{
		Domain:   "",
		Port:     0,
		Protocol: protocol,
	}

	if cluster.Status.EntryPoints == nil {
		return nil, fmt.Errorf("cluster %s does not has entrypoint", cluster.Name)
	}

	port, err := getPortFromCluster(cluster, protocol)
	if err != nil {
		return nil, fmt.Errorf("get port from cluster failed: %w", err)
	}
	entryPoint.Port = port

	domain, err := getDomainFromCluster(cluster, hostCluster)
	if err != nil {
		return nil, fmt.Errorf("get domain from cluster failed: %w", err)
	}
	entryPoint.Domain = domain

	return &entryPoint, nil
}

func getPortFromCluster(cluster v1alpha1.Cluster, protocol string) (int, error) {
	var serviceName string
	for k := range cluster.Status.EntryPoints {
		serviceName = k
		break
	}

	point := cluster.Status.EntryPoints[serviceName]

	var port int32
	switch protocol {
	case "http":
		port = point.HTTPPort
	case "https":
		port = point.HTTPSPort
	default:
		return 0, fmt.Errorf("unknow protocol %s", protocol)
	}
	return int(port), nil
}

func getDomainFromCluster(cluster v1alpha1.Cluster, hostCluster *v1alpha1.Cluster) (string, error) {
	var domain string
	if cluster.Spec.PrimaryDomain != "" {
		domain = cluster.Spec.PrimaryDomain
	} else if hostCluster != nil && hostCluster.Spec.PrimaryDomain != "" {
		domain = hostCluster.Spec.PrimaryDomain
	} else {
		var err error
		domain, err = getDomainFromURL(cluster.Spec.ApiServer)
		if err != nil {
			return "", err
		}
	}
	return domain, nil
}

func getDomainFromURL(serviceURL string) (string, error) {
	var domain string
	clusterURL, err := url.Parse(serviceURL)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(clusterURL.Host)
	if err != nil {
		return "", err
	}
	ip := net.ParseIP(host)
	if ip != nil {
		domain = fmt.Sprintf("%s.nip.io", host)
	} else {
		domain = host
	}

	return domain, nil
}
