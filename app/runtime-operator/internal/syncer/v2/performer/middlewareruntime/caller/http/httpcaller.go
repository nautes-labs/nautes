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

package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	rawhttp "net/http"
	"net/url"
	"strings"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
)

// HTTPCaller is a caller that sends HTTP requests to the middleware service provider.
type HTTPCaller struct {
	component.CallerMeta
	basicURL string
	client   rawhttp.Client
	authInfo authentication
}

type authentication struct {
	// Type is the authentication type.
	Type component.AuthType
	// token holds the access information in token format.
	token *string
}

const CallerType = "http"

func NewCaller(info component.ProviderInfo) (component.Caller, error) {
	client := &rawhttp.Client{}
	var tlsConfig *tls.Config
	var authInfo authentication

	if info.TLS != nil && info.TLS.CABundle != "" {
		caCertPool := x509.NewCertPool()

		ok := caCertPool.AppendCertsFromPEM([]byte(info.TLS.CABundle))
		if !ok {
			return nil, errors.New("failed to append CA cert")
		}

		tlsConfig = &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
	}

	if info.Auth != nil {
		switch info.Auth.Type {
		case component.AuthTypeKeypair:
			if tlsConfig == nil {
				return nil, errors.New("tls config is nil")
			}

			cert, err := tls.X509KeyPair(info.Auth.Keypair.Cert, info.Auth.Keypair.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to generate keypair: %w", err)
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
		case component.AuthTypeToken:
			authInfo.token = &info.Auth.Token.Token
		default:
			return nil, fmt.Errorf("auth type %s is not supported", info.Auth.Type)
		}
	}

	if tlsConfig != nil {
		client.Transport = &rawhttp.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return &HTTPCaller{
		CallerMeta: component.CallerMeta{
			Type:               CallerType,
			ImplementationType: component.CallerImplBasic,
		},
		basicURL: info.URL,
		client:   *client,
		authInfo: authInfo,
	}, nil
}

// Post sends a message to the remote endpoint and receives a response.
func (hc *HTTPCaller) Post(ctx context.Context, request interface{}) (result []byte, err error) {
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	req, ok := request.(*RequestHTTP)
	if !ok {
		return nil, fmt.Errorf("request is not *RequestHTTP")
	}

	query := make(url.Values)
	for k, v := range req.Query {
		query[k] = v
	}

	u, err := url.Parse(hc.basicURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	u, err = u.Parse(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	apiURL := u.ResolveReference(u).String()

	header := make(map[string][]string)
	header["Content-Type"] = []string{"application/json"}
	for k, v := range req.Header {
		header[k] = v
	}

	var body io.Reader
	if req.Body != nil {
		body = strings.NewReader(*req.Body)
	}

	httpRequest, err := rawhttp.NewRequestWithContext(ctx, req.Request, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpRequest.Header = header
	httpRequest.URL.RawQuery = query.Encode()

	if hc.authInfo.token != nil {
		httpRequest.Header.Add("Authorization", "Bearer "+*hc.authInfo.token)
	}

	httpResponse, err := hc.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send http request: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 400 {
		switch httpResponse.StatusCode {
		case 404:
			return nil, runtimeerr.ErrorResourceNotFound(errors.New(httpResponse.Status))
		case 409:
			return nil, runtimeerr.ErrorResourceExists(errors.New(httpResponse.Status))
		}
		return nil, fmt.Errorf("http request failed: %s", httpResponse.Status)
	}

	bodyBytes, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read http response body: %w", err)
	}

	return bodyBytes, nil
}

type RequestHTTP struct {
	// Request is the method of the request.
	Request string
	// Path is the path of the request.
	Path string
	// Body is the body of the request.
	Body *string
	// Header is the header of the request.
	Header map[string][]string
	// Query is the query parameters of the request.
	Query map[string][]string
}

// RequestTransformerRule stores the information of how to transform the middleware into a request.
type RequestTransformerRule struct {
	RequestGenerationRule RequestGenerationRule `json:"generation" yaml:"generation"`
	ResponseParseRule     []ResponseParseRule   `json:"parse,omitempty" yaml:"parse,omitempty"`
}

func NewRequestTransformerRule(input []byte) (*RequestTransformerRule, error) {
	dsl := &RequestTransformerRule{}
	err := json.Unmarshal(input, dsl)
	if err != nil {
		return nil, err
	}
	return dsl, nil
}

type RequestGenerationRule struct {
	// Request custom request method to use when communicating with the HTTP server
	Request string                               `json:"request" yaml:"request"`
	URI     string                               `json:"uri" yaml:"uri"`
	Header  map[string]utils.StringOrStringArray `json:"header,omitempty" yaml:"header,omitempty"`
	Query   map[string]utils.StringOrStringArray `json:"query,omitempty" yaml:"query,omitempty"`
	Body    *string                              `json:"body,omitempty" yaml:"body,omitempty"`
}

// ResponseParseRule stores the information of how to parse the response.
type ResponseParseRule struct {
	// KeyName is the key name to store the parsed variable.
	KeyName string `json:"name" yaml:"name"`
	// KeyPath is the path of the key to parse.
	KeyPath string `json:"path" yaml:"path"`
}
