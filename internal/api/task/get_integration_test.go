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

package task_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/task"
	taskGen "github.com/retr0h/osapi/internal/api/task/gen"
	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/task/client"
	"github.com/retr0h/osapi/internal/task/client/mocks"
)

type GetTaskIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *GetTaskIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *GetTaskIntegrationTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *GetTaskIntegrationTestSuite) TestGetTask() {
	tests := []struct {
		name      string
		path      string
		setupMock func() *mocks.MockManager
		wantCode  int
		wantBody  string
	}{
		{
			name: "when get Ok",
			path: "/task",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `
{
  "items": [
    {
      "body": "dGVzdCBtZXNzYWdlIDE=",
      "created": "2024-09-10T12:00:00Z",
      "id": 10
    },
    {
      "body": "dGVzdCBtZXNzYWdlIDI=",
      "created": "2024-09-10T12:00:00Z",
      "id": 11
    }
  ],
  "total_items": 2
}`,
		},
		{
			name: "when ListUndeliveredMessages errors",
			path: "/task",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().
					ListUndeliveredMessages(context.Background()).
					Return(nil, assert.AnError).
					AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when CountStreamMessages errors",
			path: "/task",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().
					ListUndeliveredMessages(context.Background()).
					Return([]client.MessageItem{}, nil).
					AnyTimes()

				mock.EXPECT().
					CountStreamMessages(context.Background()).
					Return(0, assert.AnError).
					AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0, "error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			a := api.New(suite.appConfig, suite.logger)
			taskGen.RegisterHandlers(a.Echo, task.New(mock))

			params := url.Values{}
			params.Add("limit", "2")
			params.Add("offset", "0")

			u, err := url.Parse(tc.path)
			suite.Require().NoError(err)
			u.RawQuery = params.Encode()

			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			suite.Equal(tc.wantCode, rec.Code)
			suite.JSONEq(tc.wantBody, rec.Body.String())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestGetTaskIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(GetTaskIntegrationTestSuite))
}
