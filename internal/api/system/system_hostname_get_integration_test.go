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

type SystemHostnameGetIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *SystemHostnameGetIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *SystemHostnameGetIntegrationTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *SystemHostnameGetIntegrationTestSuite) TestGetSystemHostname() {
	tests := []struct {
		name          string
		path          string
		setupHostMock func() *hostMocks.MockProvider
		wantCode      int
		wantBody      string
	}{
		{
			name: "when get Ok",
			path: "/system/hostname",
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{"hostname":"default-hostname"}`,
		},
		{
			name: "when host.GetHostname errors",
			path: "/system/hostname",
			setupHostMock: func() *hostMocks.MockProvider {
				mock := hostMocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().GetHostname().Return("", assert.AnError)
				mock.EXPECT().GetUptime().Return(time.Hour*5, nil).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			memMock := memMocks.NewPlainMockProvider(suite.ctrl)
			loadMock := loadMocks.NewPlainMockProvider(suite.ctrl)
			hostMock := tc.setupHostMock()
			diskMock := diskMocks.NewPlainMockProvider(suite.ctrl)

			a := api.New(suite.appConfig, suite.logger)
			systemGen.RegisterHandlers(a.Echo,
				system.New(
					memMock,
					loadMock,
					hostMock,
					diskMock,
				))

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
func TestSystemHostnameGetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SystemHostnameGetIntegrationTestSuite))
}
