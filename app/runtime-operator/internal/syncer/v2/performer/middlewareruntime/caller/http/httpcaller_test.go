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

package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/caller/http"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	. "github.com/onsi/ginkgo/v2"
	_ "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = component.CallerMeta{}

var _ = Describe("HTTPCaller", func() {
	var (
		httpCaller component.BasicCaller
		ctx        context.Context
		request    interface{}
	)

	BeforeEach(func() {
		httpCaller = &HTTPCaller{}
		ctx = context.Background()
	})

	Describe("NewCaller", func() {
		var (
			CAPublic = `-----BEGIN CERTIFICATE-----
MIIC/TCCAeWgAwIBAgIJAM0CEkAL+mwtMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCTEyNy4wLjAuMTAgFw0yMzAyMDEwOTU4MDlaGA8yMDUwMDYxOTA5NTgwOVow
FDESMBAGA1UEAwwJMTI3LjAuMC4xMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAn9fykf4Hk0dmEQa2zAjWytzR65oanctqiiX8L/vfstbYscPJm4B+cZkv
pq66ynn5utCF6tGlNq5CDkWLM2gohGenb3n+2fuOk25bP9o6HY4hFAEphxt3CAqs
vV0QEVoWQ+YrqWJe7Qde2hTfI33HM0mOronJPVc4LlCtaQPg6Gh1HWh7HPcxN0Wz
qIeYGenykc+I2q4CwfkZKSifu9Xgr96yJ3ySx8SNhEk4Kqyb0NHUnP/n0gVJo8IO
o/pFoiwi+MskFJzVu0Rm8oGX/GprMCc0k3Dw3l2Yn/LUsQviEl81oJZnjNZWh7WU
rLbWjhsgXsu7ERkIgCRqrKfcbqYSMwIDAQABo1AwTjAdBgNVHQ4EFgQU2eT6JVNc
4Imuo0sw3M6eNolBOpowHwYDVR0jBBgwFoAU2eT6JVNc4Imuo0sw3M6eNolBOpow
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAfOczfYn11VlDLQt6GoY
JFewHBSqFnCAWTTBfZyVH4XI5pydWuii/idwTjOv7UmVIP9EU3SOsyfCXyitQO9g
YqyiLY9qFJRxD6oXJP+M4CCdtXF9YB4akcR+8hvTG96fOSG/Ej92cXT51Hm9VCwI
Y4i1hBjnJNiJojl32hdtl79RvJ5iXk5o4u00rMz07y3t7hX/47izCb2Wo1w1Xaf5
OgD1MEnYUJdatjqb1hq+z9GDgrkc8Xb3CPCAHYk0OKij1A0YgNqrI5v7lEg1oXsx
O37YdSPd7gbdPanLOgvydTvcwuTHJLCSmmfveZHFxHrbjcEwiVzyZHetuai5lEGY
vw==
-----END CERTIFICATE-----`
			KeyPrivate = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAyTpx4uLcoKb14vIdp9WN2mvkkNwa75Nas+HZYEXb1FNXUabJ
I092xCdRnIwyaZSPGs3FWIF6RBifZaBM/1zaJprOojFb9brMU2At2Eau8/eTu0aS
nXkOsjPAKkfU8PkxNlk6tRBL2wg2niid2FvAPbN3axdQlR1Zx44IrHtQSkPsE8O5
m+EJPWDJvqZ7m0M4QJwV3tVtSOzWMaptFDQJabM+g5IJ+XGNNI7QSU7D+/k4dT+j
aT9x59LS/7PPDacLCVN0dhkPXAN7CH+gPLvriVKJPe71ye9ztBqqGtBpH5KfBzPQ
oVSoKZmxxsf89JLLfZKm99202d5uF/PhelH+SwIDAQABAoIBAQCoX9vtYbAESM/T
zo0b4yfnzIGa6GEtd5ncjCzsTmfrmLSmoK0Ke7I/3Tp/iBuilmjLn8PyE5zvn764
NVJYFiR/Sud9dVmiGmRfm0mg/zvi7ZTSjfGeDC5M09qGRkaaP5h7BlyGJpWiN5Qj
8I5q/BK2ThWtKPwHWWDHBkShtijvibqbe9QoUNkSGJ7M5lHHDdr+MzQQphZ//Vc+
GZjDH7tLv8tlWtkEC+6/4kAiiKIom2bI3e4V6m3QyfA1FQjvT3jdj63OPocKozes
btD4IMw+VrDIGRN8B7sXEE7TP7rBIkymtgg1DZUIN7ujo08AqfyuUhPkN2wAB3k/
alCJ9fohAoGBAPePifsEWYg5635KNAYz2D27/dgTX6B4OM6/jAkpOH6zkDWhe/3Y
6hgQiATXAi4cvuxljrf1wbSmyk/Y94+DBXWWtJ91u8xUtc+3AKtBCk0wnw4cPh7h
BonPyn0wReFNhA4+jwWfvh/fgtqzZcVsJCWC3nhuqdnpg1MyH8LQYLqbAoGBANAW
kDH7+WMmb2T3E8HhMIyFeef40UMk8xfZMVEKT0380zbLxSblY+w9xv8cyh1uaedH
btICWb4d3hOlnbsd2yOxcelCBNPNt7u+S4AMwgHq8W87nqAbcHpqonheXaFKtzKk
IkBecDy8vbd820w3+jZw4Cz058J14PI9yaAfFW4RAn8vYkoGwc5hRLTOd2V9ym6Z
YmIz+YFUNa6p4//pwPoPRk9T9JTHAb3M3V0rj/va16Wzmby3eVKaQVJ39g9saKei
2jW4T9CiS5SBLYXzQX+3RpcrHDzHrEqUFjGrxJGbjjq4f0Dg0rKRZzakpbHVF93T
UDlE0+muzANW6UErCLd7AoGBAJmTDXjWbogupafucjZ07E/Jct8xU8AqVP8U3MDi
ywTTw059tVOvmL+SGHvP05tFEgQPREraUUFu6ae2Y2Ll9gWxwFBW2Rk4ipGVMEOh
Js4jh2yAo+GmXqz6Zk5P1upjKjHF0UGQcWViJuJ006S8632icNC9Lw7l0M73qwbx
6e8BAoGAdnrO3gLVUmfcs8PAvtHrXEs7COFUolTRyBy5EG4LoBnxhsIahal0JVBw
kL8tqeRkFjpX4Paek2bZpiVg7dBXpW/Glhg6WBB/8tLHoXyuKfoWs8IsZclKFL/1
ENGKZpZgvBtJH4IMMEdhikUc93q5t4DLStYA8JrFT8/u1+Wc0KY=
-----END RSA PRIVATE KEY-----`
			KeyPublic = `-----BEGIN CERTIFICATE-----
MIIECTCCAvGgAwIBAgIJAPVSPX/HLe42MA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCTEyNy4wLjAuMTAgFw0yMzAyMDExMDAwMThaGA8yMDUwMDYxOTEwMDAxOFow
VTELMAkGA1UEBhMCeHgxCzAJBgNVBAgMAnh4MQswCQYDVQQHDAJ4eDELMAkGA1UE
CgwCeHgxCzAJBgNVBAsMAnh4MRIwEAYDVQQDDAkxMjcuMC4wLjEwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDJOnHi4tygpvXi8h2n1Y3aa+SQ3Brvk1qz
4dlgRdvUU1dRpskjT3bEJ1GcjDJplI8azcVYgXpEGJ9loEz/XNomms6iMVv1usxT
YC3YRq7z95O7RpKdeQ6yM8AqR9Tw+TE2WTq1EEvbCDaeKJ3YW8A9s3drF1CVHVnH
jgise1BKQ+wTw7mb4Qk9YMm+pnubQzhAnBXe1W1I7NYxqm0UNAlpsz6Dkgn5cY00
jtBJTsP7+Th1P6NpP3Hn0tL/s88NpwsJU3R2GQ9cA3sIf6A8u+uJUok97vXJ73O0
Gqoa0Gkfkp8HM9ChVKgpmbHGx/z0kst9kqb33bTZ3m4X8+F6Uf5LAgMBAAGjggEZ
MIIBFTBEBgNVHSMEPTA7gBTZ5PolU1zgia6jSzDczp42iUE6mqEYpBYwFDESMBAG
A1UEAwwJMTI3LjAuMC4xggkAzQISQAv6bC0wCQYDVR0TBAIwADALBgNVHQ8EBAMC
BDAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMIGVBgNVHREEgY0wgYqC
Cmt1YmVybmV0ZXOCEmt1YmVybmV0ZXMuZGVmYXVsdIIWa3ViZXJuZXRlcy5kZWZh
dWx0LnN2Y4Iea3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVygiRrdWJlcm5l
dGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWyHBH8AAAGHBH8AAAEwDQYJKoZI
hvcNAQELBQADggEBADKXj1/ctCEU4lMiHhQ6Ki6vI6/UflCcU5csCBSeDuRSDlLw
cx5jLj86OIJFPp3o0FNeRHm8T23RWofd5kkQMtoXzfCSTEA5HVE0pE5vXSa1pPyE
Iu1sBsO6X06tqR+ef98ioEa3ZyKNIJQ/EKBkbKa3gdq0Cvp3N330104C6JDtzJ2T
bbkWC1Ehnw/08prqZTMIix3YmSh3X73dU5NFxGkoIUjMRaqJhfRLtdYBDGFwFQon
cfZUnYiiu5Rhbe3vo8V32JfeWt3555hZMvwWYkaBuM7H5LC/0HOjCchcFDjtdvVv
Ne1AR0JiouXncOD3fUji7EjgMLTDYQR89DraG0o=
-----END CERTIFICATE-----`
		)
		It("should return a valid Caller with no TLS and Auth", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
			}
			caller, err := NewCaller(info)
			Expect(err).ToNot(HaveOccurred())
			Expect(caller).ToNot(BeNil())
		})

		It("should return an error when TLS is provided but CA bundle is invalid", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
				TLS: &component.TLSInfo{
					CABundle: "invalid",
				},
			}
			_, err := NewCaller(info)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to append CA cert"))
		})

		It("should return an error when AuthTypeKeypair is provided but tls config is nil", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
				Auth: &component.ProviderAuthInfo{
					Type: component.AuthTypeKeypair,
					Keypair: component.AuthInfoKeypair{
						Cert: []byte(KeyPublic),
						Key:  []byte(KeyPrivate),
					},
				},
			}
			_, err := NewCaller(info)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("tls config is nil"))
		})

		It("should return an error when AuthTypeKeypair is provided but keypair is invalid", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
				TLS: &component.TLSInfo{
					CABundle: CAPublic,
				},
				Auth: &component.ProviderAuthInfo{
					Type: component.AuthTypeKeypair,
					Keypair: component.AuthInfoKeypair{
						Cert: []byte("invalid"),
						Key:  []byte("invalid"),
					},
				},
			}
			_, err := NewCaller(info)
			Expect(err).To(HaveOccurred())
			Expect(strings.HasPrefix(err.Error(), "failed to generate keypair")).To(BeTrue())
		})

		It("should return an error when unsupported auth type is provided", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
				Auth: &component.ProviderAuthInfo{
					Type: "unsupported",
				},
			}
			_, err := NewCaller(info)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("auth type unsupported is not supported"))
		})

		It("should return a valid Caller with AuthTypeToken", func() {
			info := component.ProviderInfo{
				URL: "http://test.com",
				Auth: &component.ProviderAuthInfo{
					Type: component.AuthTypeToken,
					Token: component.AuthInfoToken{
						Token: "valid",
					},
				},
			}
			caller, err := NewCaller(info)
			Expect(err).ToNot(HaveOccurred())
			Expect(caller).ToNot(BeNil())
		})
	})

	Describe("Post", func() {
		Context("when request is nil", func() {
			BeforeEach(func() {
				request = nil
			})

			It("should return an error", func() {
				_, err := httpCaller.Post(ctx, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("request is nil"))
			})
		})

		Context("when request is RequestHTTP", func() {
			var (
				server  *httptest.Server
				handler http.HandlerFunc
			)

			BeforeEach(func() {
				request = &RequestHTTP{
					Request: "POST",
					Path:    "/test",
					Body:    new(string),
					Query: map[string][]string{
						"key": {"value"},
					},
				}

				handler = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}

				server = httptest.NewServer(http.HandlerFunc(handler))

				authInfo := component.AuthInfoToken{
					Token: "test-token",
				}

				providerAuthInfo, err := component.NewAuthInfo(authInfo)
				if err != nil {
					panic(err)
				}

				providerInfo := component.ProviderInfo{
					Type: "Test Provider",
					URL:  server.URL,
					Auth: providerAuthInfo,
				}

				caller, err := NewCaller(providerInfo)
				if err != nil {
					panic(err)
				}
				httpCaller = caller.(component.BasicCaller)
			})

			AfterEach(func() {
				server.Close()
			})

			It("when response status code is >= 400, should return an error", func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
				server.Config.Handler = http.HandlerFunc(handler)
				_, err := httpCaller.Post(context.Background(), request)
				Expect(err).To(HaveOccurred())
			})

			It("when response status code is < 400, should return the response", func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				}
				server.Config.Handler = http.HandlerFunc(handler)
				response, err := httpCaller.Post(context.Background(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(Equal([]byte("OK")))
			})
		})
	})
})

var _ = Describe("NewRequestTransformerRule", func() {
	It("should return a valid RequestTransformerRule", func() {
		input := []byte(`{"Generation": {"Request": "GET", "URI": "/test"}}`)
		rule, err := NewRequestTransformerRule(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(rule).ToNot(BeNil())
		Expect(rule.RequestGenerationRule.Request).To(Equal("GET"))
		Expect(rule.RequestGenerationRule.URI).To(Equal("/test"))
	})

	It("should return an error for invalid input", func() {
		input := []byte(`{"key":}`)
		_, err := NewRequestTransformerRule(input)
		Expect(err).To(HaveOccurred())
	})

	It("should parse ResponseParseRule correctly", func() {
		input := []byte(`{
            "Generation": {"Request": "GET", "URI": "/test"},
            "Parse": [{"Name": "testName", "Path": "testPath"}]
        }`)
		rule, err := NewRequestTransformerRule(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(rule).ToNot(BeNil())
		Expect(rule.ResponseParseRule).To(HaveLen(1))
		Expect(rule.ResponseParseRule[0].KeyName).To(Equal("testName"))
		Expect(rule.ResponseParseRule[0].KeyPath).To(Equal("testPath"))
	})
})
