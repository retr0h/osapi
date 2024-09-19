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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/network"
	networkGen "github.com/retr0h/osapi/internal/api/network/gen"
	"github.com/retr0h/osapi/internal/config"
	dnsMocks "github.com/retr0h/osapi/internal/provider/dns/mocks"
	networkMocks "github.com/retr0h/osapi/internal/provider/network/mocks"
)

type NetworkDNSIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *NetworkDNSIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *NetworkDNSIntegrationTestSuite) TestGetNetworkDNS() {
	tests := []struct {
		name      string
		path      string
		setupMock func() *dnsMocks.MockProvider
		wantCode  int
		wantBody  string
	}{
		{
			name: "when get ok",
			path: "/network/dns",
			setupMock: func() *dnsMocks.MockProvider {
				mock := dnsMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{
"search_domains": [
  "example.com",
  "local.lan"
],
"servers": [
  "192.168.1.1",
  "8.8.8.8",
  "8.8.4.4",
  "2001:4860:4860::8888",
  "2001:4860:4860::8844"
]}`,
		},
		{
			name: "when GetResolvConf errors",
			path: "/network/dns",
			setupMock: func() *dnsMocks.MockProvider {
				mock := dnsMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetResolvConf().
					Return(nil, fmt.Errorf("GetResolvConf error")).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0,"error":"GetResolvConf error"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			networkMock := networkMocks.NewDefaultMockProvider(suite.ctrl)
			dnsMock := tc.setupMock()

			a := api.New(suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a.Echo, network.New(networkMock, dnsMock))

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)
			assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestNetworkDNSIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkDNSIntegrationTestSuite))
}
