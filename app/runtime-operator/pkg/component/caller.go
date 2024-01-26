// Copyright 2024 Nautes Authors
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

package component

import (
	"context"
	"fmt"
)

// ProviderInfo stores detailed information about the middleware service provider. This information is used to interact with the provider's API.
type ProviderInfo struct {
	// Type is the name of the middleware service provider.
	Type string
	// URL is the base URL of the middleware service provider's API. All API requests will be sent to this address.
	URL string
	// TLS stores the TLS configuration information used for secure communication with the middleware provider.
	TLS *TLSInfo
	// Auth stores the authentication information used to access the middleware provider's API. This may include API keys, OAuth tokens, etc.
	Auth *ProviderAuthInfo
}

type TLSInfo struct {
	CABundle string
}

// ProviderAuthInfo represents the authentication information for a provider.
type ProviderAuthInfo struct {
	// Type is the authentication type.
	Type AuthType
	// Keypair holds the access information in keypair format.
	Keypair AuthInfoKeypair
	// Token holds the access information in token format.
	Token AuthInfoToken
	// UserPassword holds the access information in username/password format.
	UserPassword AuthInfoUserPassword
}

// NewAuthInfo creates a new ProviderAuthInfo object based on the provided authInfo.
// It determines the type of authInfo and sets the corresponding fields in the ProviderAuthInfo object.
// If the authInfo type is unknown, it returns an error.
func NewAuthInfo(authInfo interface{}) (*ProviderAuthInfo, error) {
	info := &ProviderAuthInfo{}
	switch authInfo := authInfo.(type) {
	case AuthInfoKeypair:
		info.Type = AuthTypeKeypair
		info.Keypair = authInfo
	case AuthInfoToken:
		info.Type = AuthTypeToken
		info.Token = authInfo
	case AuthInfoUserPassword:
		info.Type = AuthTypeUserPassword
		info.UserPassword = authInfo
	default:
		return nil, fmt.Errorf("unknown auth info type: %T", authInfo)
	}
	return info, nil
}

// AuthInfoKeypair represents a key pair used for authentication.
type AuthInfoKeypair struct {
	// Key is the private key of the key pair.
	Key []byte
	// Cert is the public key of the key pair.
	Cert []byte
}

// AuthInfoToken represents authentication information token.
type AuthInfoToken struct {
	Token string // Token is the authorization token.
}

// AuthInfoUserPassword represents the authentication information for a user with a password.
type AuthInfoUserPassword struct {
	Username string // The username of the user.
	Password string // The password of the user.
}

// CallerImplementationType represents the type of caller implementation.
type CallerImplementationType string

const (
	// CallerImplBasic represents the basic caller implementation type.
	CallerImplBasic CallerImplementationType = "Basic"
	// CallerImplAdvanced represents the advanced caller implementation type.
	CallerImplAdvanced CallerImplementationType = "Advanced"
)

// CallerMeta is the metadata information of Caller.
type CallerMeta struct {
	// Type is the type of the Caller.
	Type string
	// ImplementationType is the level of the Caller, whether it is BasicCaller or AdvancedCaller.
	ImplementationType CallerImplementationType
}

// GetType returns the type of the caller.
func (cm CallerMeta) GetType() string {
	return cm.Type
}

// GetImplementationType returns the implementation type of the caller.
func (cm CallerMeta) GetImplementationType() CallerImplementationType {
	return cm.ImplementationType
}

// Caller is a caller that can send messages to a remote endpoint and receive messages from it.
type Caller interface {
	GetType() string
	GetImplementationType() CallerImplementationType
}

// BasicCaller is a type of primitive Caller. It is responsible for sending messages to the remote endpoint.
// It cannot understand the messages it sends or receives. Additional conversion tools are needed to interpret the results.
// Examples include: http, ssh, client-go
type BasicCaller interface {
	Caller
	// Post sends a message to the remote endpoint and receives a response.
	// Parameters:
	// - request: The content to be sent. The specific data format is defined by the implementation of the Caller.
	// Returns:
	// - result: The response from the remote endpoint after sending the message.
	// - err: An error message if the sending fails.
	Post(ctx context.Context, request interface{}) (result []byte, err error)
}

// AdvancedCaller is an advanced type of Caller. It can recognize resource types on its own and requires complete
// declaration information when invoked. It can sort itself and has its own caching format.
// Examples include: terraform, ansible (when deploying only using roles)
type AdvancedCaller interface {
	Caller
	// Deploy synchronizes resources to the remote endpoint according to the format of the resource declaration.
	// Parameters:
	// - resources: The collection of resources to be deployed.
	// - state: The cache returned from the previous deployment.
	// - opts: Parameters for the sending action itself. For example, when using the HTTP protocol, the method and address of the request.
	// Returns:
	// - newState: The state information after the deployment.
	// - err: An error message if the sending fails.
	Deploy(ctx context.Context, resources string, state interface{}, opts map[string]interface{}) (newState interface{}, err error)
	// Delete cleans up the deployed resources on the remote endpoint based on the provided cache information.
	Delete(ctx context.Context, state interface{}) error
}

const (
	CallerTypeHTTP = "http"
)

// callerFactory is a factoryCaller struct that holds a menu of NewCaller functions.
var callerFactory = factoryCaller{
	// menu is a map of NewCaller functions. The key is the name of the caller type.
	menu: map[string]NewCaller{},
}

// NewCaller is a function type that takes a ProviderInfo parameter and returns a Caller and an error.
type NewCaller func(providerInfo ProviderInfo) (Caller, error)

// callerFactory represents a factory for creating callers.
type factoryCaller struct {
	menu map[string]NewCaller
}

// AddFunctionNewCaller adds a new function to the CallerFactory menu.
// The function is identified by the given name and is associated with the provided NewCaller function.
func AddFunctionNewCaller(name string, fn NewCaller) {
	callerFactory.menu[name] = fn
}

// GetCaller retrieves an instance of the caller based on the name and provider information.
// If the corresponding caller is not found, an error is returned.
func GetCaller(name string, providerInfo ProviderInfo) (Caller, error) {
	fn, ok := callerFactory.menu[name]
	if !ok {
		return nil, fmt.Errorf("caller %s not found", name)
	}
	return fn(providerInfo)
}

func ClearCallerFactory() {
	callerFactory.menu = map[string]NewCaller{}
}
