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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/network"
	networkGen "github.com/retr0h/osapi/internal/api/network/gen"
	"github.com/retr0h/osapi/internal/config"
	networkProvider "github.com/retr0h/osapi/internal/provider/network"
)

type NetworkDNSIntegrationTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *NetworkDNSIntegrationTestSuite) SetupTest() {
	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *NetworkDNSIntegrationTestSuite) TestGetNetworkDNS() {
	tests := []struct {
		name      string
		path      string
		setupMock func() *networkProvider.Mock
		wantCode  int
		wantBody  string
	}{
		{
			name: "when get ok",
			path: "/network/dns",
			setupMock: func() *networkProvider.Mock {
				mock := networkProvider.NewDefaultMock()
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
			setupMock: func() *networkProvider.Mock {
				mock := networkProvider.NewDefaultMock()
				mock.GetResolvConfFunc = func() (*networkProvider.DNSConfig, error) {
					return nil, fmt.Errorf("GetResolvConf error")
				}
				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{}`,
		},
		{
			name: "when endpoint found",
			path: "/network/notfound",
			setupMock: func() *networkProvider.Mock {
				mock := networkProvider.NewDefaultMock()
				return mock
			},
			wantCode: http.StatusNotFound,
			wantBody: `{}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			a := api.New(suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a.Echo, network.New(mock))

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)

			if tc.wantCode == http.StatusOK {
				assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestNetworkDNSIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkDNSIntegrationTestSuite))
}