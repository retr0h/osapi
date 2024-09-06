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

	"github.com/spf13/afero"
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

	appFs     afero.Fs
	appConfig config.Config
	logger    *slog.Logger
}

func (suite *NetworkDNSIntegrationTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *NetworkDNSIntegrationTestSuite) TestGetNetworkDNS() {
	tests := []struct {
		name      string
		uri       string
		setupMock func() *networkProvider.MockNetwork
		want      struct {
			code int
			body string
		}
	}{
		{
			name: "when http ok",
			uri:  "/network/dns",
			setupMock: func() *networkProvider.MockNetwork {
				mock := networkProvider.NewDefaultMockNetwork()
				return mock
			},
			want: struct {
				code int
				body string
			}{
				code: http.StatusOK,
				body: `{
"searchDomains": [
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
		},
		{
			name: "when GetResolvConf errors",
			uri:  "/network/dns",
			setupMock: func() *networkProvider.MockNetwork {
				mock := networkProvider.NewDefaultMockNetwork()
				mock.GetResolvConfFunc = func() (*networkProvider.DNSConfig, error) {
					return nil, fmt.Errorf("GetResolvConf error")
				}
				return mock
			},
			want: struct {
				code int
				body string
			}{
				code: http.StatusInternalServerError,
				body: `{}`,
			},
		},
		{
			name: "when not found",
			uri:  "/network/notfound",
			setupMock: func() *networkProvider.MockNetwork {
				mock := networkProvider.NewDefaultMockNetwork()
				return mock
			},
			want: struct {
				code int
				body string
			}{
				code: http.StatusNotFound,
				body: `{}`,
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			a := api.New(suite.appFs, suite.appConfig, suite.logger)
			networkGen.RegisterHandlers(a, network.New(mock))

			// Create a new request to the network endpoint
			req := httptest.NewRequest(http.MethodGet, tc.uri, nil)
			rec := httptest.NewRecorder()

			// Serve the request
			a.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.want.code, rec.Code)

			if tc.want.code == http.StatusOK {
				assert.JSONEq(suite.T(), tc.want.body, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestNetworkDNSIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkDNSIntegrationTestSuite))
}
