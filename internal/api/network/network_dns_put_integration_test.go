// Copyright (c) 2024 John Dewey

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

package network_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/network"
	networkGen "github.com/retr0h/osapi/internal/api/network/gen"
	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/provider/network/dns/mocks"
	pingMocks "github.com/retr0h/osapi/internal/provider/network/ping/mocks"
	queueMocks "github.com/retr0h/osapi/internal/queue/mocks"
)

type NetworkDNSPutIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *NetworkDNSPutIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *NetworkDNSPutIntegrationTestSuite) TestPutNetworkDNS() {
	tests := []struct {
		name           string
		path           string
		body           string
		setupMock      func() *mocks.MockProvider
		setupQueueMock func() *queueMocks.MockManager
		wantCode       int
		wantBody       string
	}{
		{
			name: "when put Ok",
			path: "/network/dns",
			body: `{
"servers": [ "1.1.1.1", "2001:4860:4860::8888"],
"search_domains": [ "foo.bar"]
}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusAccepted,
			wantBody: `{"id":"12345"}`,
		},
		{
			name: "when put Ok with missing body Servers",
			path: "/network/dns",
			body: `{
"search_domains": [ "foo.bar"]
}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusAccepted,
			wantBody: `{"id":"12345"}`,
		},
		{
			name: "when put Ok with missing body Search Domains",
			path: "/network/dns",
			body: `{
"servers": [ "1.1.1.1", "2001:4860:4860::8888"]
}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusAccepted,
			wantBody: `{"id":"12345"}`,
		},
		{
			name: "when body is empty",
			path: "/network/dns",
			body: `{}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'DNSConfigUpdate.SearchDomains' Error:Field validation for 'SearchDomains' failed on the 'required_without' tag\nKey: 'DNSConfigUpdate.Servers' Error:Field validation for 'Servers' failed on the 'required_without' tag"}`,
		},
		{
			name: "when body's Servers are not a proper ipv4 and ipv6 addresses",
			path: "/network/dns",
			body: `{
"servers": [ "1.1", "2001:4860:4860:8888"]
}`, // Invalid ipv4 and ipv6 addresses
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'DNSConfigUpdate.Servers[0]' Error:Field validation for 'Servers[0]' failed on the 'ip' tag\nKey: 'DNSConfigUpdate.Servers[1]' Error:Field validation for 'Servers[1]' failed on the 'ip' tag"}`,
		},
		{
			name: "when body's Search Domains are invalid",
			path: "/network/dns",
			body: `{
"search_domains": [ "example..com", "-example.com", "example-.com", "excample_123.com"]
}`, // Invalid RFC 1123 hostnames
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'DNSConfigUpdate.SearchDomains[0]' Error:Field validation for 'SearchDomains[0]' failed on the 'hostname' tag\nKey: 'DNSConfigUpdate.SearchDomains[1]' Error:Field validation for 'SearchDomains[1]' failed on the 'hostname' tag\nKey: 'DNSConfigUpdate.SearchDomains[3]' Error:Field validation for 'SearchDomains[3]' failed on the 'hostname' tag"}`,
		},
		{
			name: "when body is malformed",
			path: "/network/dns",
			body: `{"body": }`, // Malformed JSON
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"code=400, message=Syntax error: offset=10, error=invalid character '}' looking for beginning of value, internal=invalid character '}' looking for beginning of value"}`,
		},
		{
			name: "when task.CreateAndMarshalChangeDNSAction errors",
			path: "/network/dns",
			body: `{
"servers": [ "1.1.1.1", "2001:4860:4860::8888"],
"search_domains": [ "foo.bar"]
}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				network.CreateAndMarshalChangeDNSActionFunc = func(servers []string, searchDomains []string) ([]byte, error) {
					return nil, assert.AnError
				}
				mock := queueMocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when queue.Put errors",
			path: "/network/dns",
			body: `{
"servers": [ "1.1.1.1", "2001:4860:4860::8888"],
"search_domains": [ "foo.bar"]
}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupQueueMock: func() *queueMocks.MockManager {
				network.CreateAndMarshalChangeDNSActionFunc = func(servers []string, searchDomains []string) ([]byte, error) {
					return []byte("mocked-dns-action"), nil
				}

				mock := queueMocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().Put(context.Background(), []byte("mocked-dns-action")).
					Return(assert.AnError).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			pingMock := pingMocks.NewDefaultMockProvider(suite.ctrl)
			dnsMock := tc.setupMock()
			queueMock := tc.setupQueueMock()

			a := api.New(suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a.Echo, network.New(pingMock, dnsMock, queueMock))

			req := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)
			assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestNetworkDNSPutIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkDNSPutIntegrationTestSuite))
}
