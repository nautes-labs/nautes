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

package v1alpha1

// GetProduct returns the product name associated with the MiddlewareRuntime.
func (r *MiddlewareRuntime) GetProduct() string {
	return r.Namespace
}

// GetDestination returns the destination environment of the MiddlewareRuntime.
func (r *MiddlewareRuntime) GetDestination() string {
	return r.Spec.Destination.Environment
}

// GetNamespaces returns a list of namespaces associated with the MiddlewareRuntime.
// If the MiddlewareRuntime's Spec.Destination.Space is not empty, it is added to the list.
// Otherwise, the MiddlewareRuntime's name is added to the list.
// Additionally, any namespaces associated with the MiddlewareRuntime's middlewares are also added to the list.
func (r *MiddlewareRuntime) GetNamespaces() []string {
	var namespaces []string
	if len(r.Spec.Destination.Space) != 0 {
		namespaces = append(namespaces, r.Spec.Destination.Space)
	} else {
		namespaces = append(namespaces, r.Name)
	}

	for _, middleware := range r.GetMiddlewares() {
		if len(middleware.Space) != 0 {
			namespaces = append(namespaces, middleware.Space)
		}
	}

	return namespaces
}

// GetRuntimeType returns the type of the middleware runtime.
func (r *MiddlewareRuntime) GetRuntimeType() RuntimeType {
	return RuntimeTypeDeploymentRuntime
}

// GetAccount returns the account associated with the MiddlewareRuntime.
// If the account is not specified in the specification, it returns the name of the MiddlewareRuntime.
// Otherwise, it returns the specified account.
func (r *MiddlewareRuntime) GetAccount() string {
	if r.Spec.Account == "" {
		return r.GetName()
	}
	return r.Spec.Account
}

// GetMiddlewares returns a slice of Middleware objects that represent all the middlewares
// defined in the MiddlewareRuntime instance.
func (r *MiddlewareRuntime) GetMiddlewares() []Middleware {
	var middlewares []Middleware

	middlewares = append(middlewares, r.Spec.DataBases...)
	middlewares = append(middlewares, r.Spec.Monitors...)
	middlewares = append(middlewares, r.Spec.Storages...)
	middlewares = append(middlewares, r.Spec.SecureStores...)

	return middlewares
}

// GetDeployment returns the deployment information for the middleware.
// It checks the type of the middleware and returns the corresponding deployment.
// If the middleware type is Redis, it returns the Redis deployment.
// Otherwise, it returns the generic middleware information.
func (m Middleware) GetDeployment() interface{} {
	switch m.Type {
	case MiddlewareTypeRedis:
		return m.Redis
	default:
		return m.CommonMiddlewareInfo
	}
}
