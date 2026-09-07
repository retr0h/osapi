// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package client_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
)

type DNSPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *DNSPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *DNSPublicTestSuite) TestGet() {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		serverURL     string
		target        string
		interfaceName string
		validateFunc  func(*client.Response[client.Collection[client.DNSConfig]], error)
	}{
		{
			name:          "when requesting DNS returns results",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(`{"results":[{"hostname":"dns-host","servers":["8.8.8.8"]}]}`),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSConfig]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("dns-host", resp.Data.Results[0].Hostname)
				suite.Equal([]string{"8.8.8.8"}, resp.Data.Results[0].Servers)
			},
		},
		{
			name:          "when server returns 403 returns AuthError",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSConfig]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:          "when client HTTP call fails returns error",
			target:        "_any",
			interfaceName: "eth0",
			serverURL:     "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Collection[client.DNSConfig]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "get dns")
			},
		},
		{
			name:          "when server returns 200 with no JSON body returns UnexpectedStatusError",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSConfig]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.DNS.Get(suite.ctx, tc.target, tc.interfaceName)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DNSPublicTestSuite) TestUpdate() {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		serverURL     string
		target        string
		iface         string
		servers       []string
		searchDomains []string
		overrideDHCP  bool
		validateFunc  func(*client.Response[client.Collection[client.DNSUpdateResult]], error)
	}{
		{
			name:          "when servers only provided sets servers",
			target:        "_any",
			iface:         "eth0",
			servers:       []string{"8.8.8.8", "8.8.4.4"},
			searchDomains: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"completed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("dns-host", resp.Data.Results[0].Hostname)
				suite.Equal("completed", resp.Data.Results[0].Status)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name:          "when search domains only provided sets search domains",
			target:        "_any",
			iface:         "eth0",
			servers:       nil,
			searchDomains: []string{"example.com"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"completed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name:          "when both provided sets servers and search domains",
			target:        "_any",
			iface:         "eth0",
			servers:       []string{"8.8.8.8"},
			searchDomains: []string{"example.com"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"completed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name:         "when override DHCP provided sets override_dhcp",
			target:       "_any",
			iface:        "eth0",
			servers:      []string{"8.8.8.8"},
			overrideDHCP: true,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"completed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name:          "when neither provided sends empty body",
			target:        "_any",
			iface:         "eth0",
			servers:       nil,
			searchDomains: nil,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"completed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name:    "when server returns 403 returns AuthError",
			target:  "_any",
			iface:   "eth0",
			servers: []string{"8.8.8.8"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			target:    "_any",
			iface:     "eth0",
			servers:   []string{"8.8.8.8"},
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "update dns")
			},
		},
		{
			name:    "when server returns 202 with no JSON body returns UnexpectedStatusError",
			target:  "_any",
			iface:   "eth0",
			servers: []string{"8.8.8.8"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.DNSUpdateResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.DNS.Update(
				suite.ctx,
				tc.target,
				tc.iface,
				tc.servers,
				tc.searchDomains,
				tc.overrideDHCP,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DNSPublicTestSuite) TestDelete() {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		serverURL     string
		target        string
		interfaceName string
		validateFunc  func(*client.Response[client.Collection[client.DNSDeleteResult]], error)
	}{
		{
			name:          "when deleting DNS returns results",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"dns-host","status":"ok","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DNSDeleteResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("dns-host", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name:          "when server returns 403 returns AuthError",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DNSDeleteResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:          "when server returns 400 returns ValidationError",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DNSDeleteResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:          "when client HTTP call fails returns error",
			target:        "_any",
			interfaceName: "eth0",
			serverURL:     "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DNSDeleteResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "delete dns")
			},
		},
		{
			name:          "when server returns 200 with no JSON body returns UnexpectedStatusError",
			target:        "_any",
			interfaceName: "eth0",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DNSDeleteResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.DNS.Delete(suite.ctx, tc.target, tc.interfaceName)
			tc.validateFunc(resp, err)
		})
	}
}

func TestDNSPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DNSPublicTestSuite))
}
