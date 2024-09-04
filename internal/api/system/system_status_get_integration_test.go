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

package system_test

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
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/config"
	systemProvider "github.com/retr0h/osapi/internal/provider/system"
)

type SystemStatusIntegrationTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *SystemStatusIntegrationTestSuite) SetupTest() {
	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *SystemStatusIntegrationTestSuite) TestGetSystemStatus() {
	tests := []struct {
		name      string
		uri       string
		setupMock func() *systemProvider.MockSystem
		want      struct {
			code int
			body string
		}
	}{
		{
			name: "when http ok",
			uri:  "/system/status",
			setupMock: func() *systemProvider.MockSystem {
				mock := systemProvider.NewDefaultMockSystem()
				return mock
			},
			want: struct {
				code int
				body string
			}{
				code: http.StatusOK,
				body: `{
"disk": {
    "free": 0,
    "total": 0,
    "used": 0
},
"hostname": "default-hostname",
"load_average": {
    "15min": 0.2,
    "1min": 1,
    "5min": 0.5
},
"memory": {
    "free": 4.194304e+06,
    "total": 8.388608e+06,
    "used": 2.097152e+06
},
"uptime": "0 days, 5 hours, 0 minutes"
}`,
			},
		},
		{
			name: "when get hostname errors",
			uri:  "/system/status",
			setupMock: func() *systemProvider.MockSystem {
				mock := systemProvider.NewDefaultMockSystem()
				mock.GetHostnameFunc = func() (string, error) {
					return "", fmt.Errorf("GetHostname error")
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
			uri:  "/system/notfound",
			setupMock: func() *systemProvider.MockSystem {
				mock := systemProvider.NewDefaultMockSystem()
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
			a := api.New(suite.appConfig, suite.logger)
			systemGen.RegisterHandlers(a, system.New(mock))

			// Create a new request to the /system/status endpoint
			req := httptest.NewRequest(http.MethodGet, tc.uri, nil)
			rec := httptest.NewRecorder()

			// Serve the request
			a.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.want.code, rec.Code)

			fmt.Println(rec.Body.String())
			if tc.want.code == http.StatusOK {
				assert.JSONEq(suite.T(), tc.want.body, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestSystemStatusIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStatusIntegrationTestSuite))
}
