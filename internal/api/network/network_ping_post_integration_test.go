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
	dnsMocks "github.com/retr0h/osapi/internal/provider/network/dns/mocks"
	"github.com/retr0h/osapi/internal/provider/network/ping/mocks"
	queueMocks "github.com/retr0h/osapi/internal/queue/mocks"
)

type NetworkPingPostIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger

	okBody string
}

func (suite *NetworkPingPostIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	suite.okBody = `
{
  "avg_rtt":"15ms",
  "max_rtt":"20ms",
  "min_rtt":"10ms",
  "packet_loss":0,
  "packets_received":3,
  "packets_sent":3
}`
}

func (suite *NetworkPingPostIntegrationTestSuite) TestPostNetworkPing() {
	tests := []struct {
		name      string
		path      string
		body      string
		setupMock func() *mocks.MockProvider
		wantCode  int
		wantBody  string
	}{
		{
			name: "when post Ok",
			path: "/network/ping",
			body: `{"address": "1.1.1.1"}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: suite.okBody,
		},
		{
			name: "when post Ok with ipv6 body Address",
			path: "/network/ping",
			body: `{"address": "2001:4860:4860::8888"}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: suite.okBody,
		},
		{
			name: "when body is empty",
			path: "/network/ping",
			body: `{}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'PostNetworkPingJSONBody.Address' Error:Field validation for 'Address' failed on the 'required' tag"}`,
		},
		{
			name: "when body's Address is not a proper ipv4 address",
			path: "/network/ping",
			body: `{"address": "1.1."}`, // Missing last octets
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'PostNetworkPingJSONBody.Address' Error:Field validation for 'Address' failed on the 'ip' tag"}`,
		},
		{
			name: "when body's Address is not a proper ipv6 address",
			path: "/network/ping",
			body: `{"address": "2001:4860:4860:8888"}`, // Missing segments, no "::" for compression
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Key: 'PostNetworkPingJSONBody.Address' Error:Field validation for 'Address' failed on the 'ip' tag"}`,
		},
		{
			name: "when body is malformed",
			path: "/network/ping",
			body: `{"body": }`, // Malformed JSON
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"code=400, message=Syntax error: offset=10, error=invalid character '}' looking for beginning of value, internal=invalid character '}' looking for beginning of value"}`,
		},
		{
			name: "when ping.Do errors",
			path: "/network/ping",
			body: `{"address": "1.1.1.1"}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().Do("1.1.1.1").
					Return(nil, assert.AnError).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			pingMock := tc.setupMock()
			dnsMock := dnsMocks.NewDefaultMockProvider(suite.ctrl)
			queueMock := queueMocks.NewDefaultMockManager(suite.ctrl)

			a := api.New(suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a.Echo, network.New(pingMock, dnsMock, queueMock))

			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
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
func TestNetworkPingPostIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkPingPostIntegrationTestSuite))
}
