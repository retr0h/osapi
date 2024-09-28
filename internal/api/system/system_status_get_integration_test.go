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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/config"
	hostnameMocks "github.com/retr0h/osapi/internal/provider/system/hostname/mocks"
	loadMocks "github.com/retr0h/osapi/internal/provider/system/load/mocks"
	memMocks "github.com/retr0h/osapi/internal/provider/system/mem/mocks"
	"github.com/retr0h/osapi/internal/provider/system/mocks"
)

type SystemStatusIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *SystemStatusIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *SystemStatusIntegrationTestSuite) TestGetSystemStatus() {
	tests := []struct {
		name              string
		path              string
		setupMock         func() *mocks.MockProvider
		setupHostnameMock func() *hostnameMocks.MockProvider
		setupMemMock      func() *memMocks.MockProvider
		setupLoadMock     func() *loadMocks.MockProvider
		wantCode          int
		wantBody          string
	}{
		{
			name: "when get ok",
			path: "/system/status",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostnameMock: func() *hostnameMocks.MockProvider {
				mock := hostnameMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{
"disks": [{
  "free":250000000000,
  "name":"/dev/disk1",
  "total":500000000000,
  "used":250000000000
}],
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
		{
			name: "when hostname.Get errors",
			path: "/system/status",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostnameMock: func() *hostnameMocks.MockProvider {
				mock := hostnameMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().Get().Return("", assert.AnError)

				return mock
			},
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
		// uptime, err := s.SystemProvider.GetUptime()
		{
			name: "when load.GetAverageStats errors",
			path: "/system/status",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostnameMock: func() *hostnameMocks.MockProvider {
				mock := hostnameMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetAverageStats().Return(nil, assert.AnError)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when mem.GetStats errors",
			path: "/system/status",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupHostnameMock: func() *hostnameMocks.MockProvider {
				mock := hostnameMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			setupMemMock: func() *memMocks.MockProvider {
				mock := memMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetStats().Return(nil, assert.AnError)

				return mock
			},
			setupLoadMock: func() *loadMocks.MockProvider {
				mock := loadMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
		// diskStats, err := s.SystemProvider.GetLocalDiskStats()
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			hostnameMock := tc.setupHostnameMock()
			memMock := tc.setupMemMock()
			loadMock := tc.setupLoadMock()

			a := api.New(suite.appConfig, suite.logger)
			systemGen.RegisterHandlers(a.Echo,
				system.New(mock,
					hostnameMock,
					memMock,
					loadMock,
				))

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
func TestSystemStatusIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStatusIntegrationTestSuite))
}
