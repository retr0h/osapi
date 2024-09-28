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
)

type NetworkPingIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *NetworkPingIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *NetworkPingIntegrationTestSuite) TestGetNetworkDNS() {
	tests := []struct {
		name      string
		path      string
		body      string
		setupMock func() *mocks.MockProvider
		wantCode  int
		wantBody  string
	}{
		{
			name: "when get ok",
			path: "/network/ping",
			body: `{"address": "example.com"}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{
"avg_rtt":"15ms",
"max_rtt":"20ms",
"min_rtt":"10ms",
"packet_loss":0,
"packets_received":3,
"packets_sent":3
      }`,
		},
		{
			name: "when body is missing Address",
			path: "/network/ping",
			body: `{}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Address field is required and cannot be empty"}`,
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
			body: `{"address": "example.com"}`,
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().Do("example.com").
					Return(nil, assert.AnError).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			networkMock := tc.setupMock()
			dnsMock := dnsMocks.NewDefaultMockProvider(suite.ctrl)

			a := api.New(suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a.Echo, network.New(networkMock, dnsMock))

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
func TestNetworkPingIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkPingIntegrationTestSuite))
}
