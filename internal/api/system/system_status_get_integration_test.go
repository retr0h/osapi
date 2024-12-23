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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/config"
	diskMocks "github.com/retr0h/osapi/internal/provider/system/disk/mocks"
	hostMocks "github.com/retr0h/osapi/internal/provider/system/host/mocks"
	loadMocks "github.com/retr0h/osapi/internal/provider/system/load/mocks"
	memMocks "github.com/retr0h/osapi/internal/provider/system/mem/mocks"
)

type SystemStatusGetIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *SystemStatusGetIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *SystemStatusGetIntegrationTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *SystemStatusGetIntegrationTestSuite) TestGetSystemStatus() {
	tests := []struct {
		name          string
		path          string
		setupMemMock  func() *memMocks.MockProvider
		setupLoadMock func() *loadMocks.MockProvider
		setupHostMock func() *hostMocks.MockProvider
		setupDiskMock func() *diskMocks.MockProvider
		wantCode      int
		wantBody      string
	}{
		{
			name: "when get Ok",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `
{
  "disks": [
    {
      "free": 250000000000,
      "name": "/dev/disk1",
      "total": 500000000000,
      "used": 250000000000
    }
  ],
  "hostname": "default-hostname",
  "load_average": {
    "1min": 1,
    "5min": 0.5,
    "15min": 0.2
  },
  "memory": {
    "free": 4194304,
    "total": 8388608,
    "used": 2097152
  },
  "os_info": {
    "distribution": "Ubuntu",
    "version": "24.04"
  },
  "uptime": "0 days, 5 hours, 0 minutes"
}
`,
		},
		{
			name: "when host.GetHostname errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetHostname().Return("", assert.AnError)
				mock.EXPECT().GetUptime().Return(time.Hour*5, nil).AnyTimes()

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when host.GetUptime errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetHostname().Return("default-hostname", nil).AnyTimes()
				mock.EXPECT().GetUptime().Return(0*time.Second, assert.AnError)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when load.GetAverageStats errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetAverageStats().Return(nil, assert.AnError)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when mem.GetStats errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetStats().Return(nil, assert.AnError)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when host.GetOSInfo errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetHostname().Return("default-hostname", nil).AnyTimes()
				mock.EXPECT().GetUptime().Return(time.Hour*5, nil).AnyTimes()
				mock.EXPECT().GetOSInfo().Return(nil, assert.AnError)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when disk.GetLocalUsageStats errors",
			path: "/system/status",
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupDiskMock: func() *diskMocks.MockProvider {
				mock := diskMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetLocalUsageStats().Return(nil, assert.AnError)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			memMock := tc.setupMemMock()
			loadMock := tc.setupLoadMock()
			hostMock := tc.setupHostMock()
			diskMock := tc.setupDiskMock()

			systemHandler := system.New(memMock, loadMock, hostMock, diskMock)
			strictHandler := systemGen.NewStrictHandler(systemHandler, nil)

			a := api.New(suite.appConfig, suite.logger)
			systemGen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			suite.Equal(tc.wantCode, rec.Code)
			suite.JSONEq(tc.wantBody, rec.Body.String())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestSystemStatusGetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStatusGetIntegrationTestSuite))
}
