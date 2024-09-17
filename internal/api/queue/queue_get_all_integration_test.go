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

package queue_test

import (
	"context"
	"fmt"
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
	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/queue/mocks"
)

type QueueGetAllIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *QueueGetAllIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.appConfig = config.Config{
		Queue: config.Queue{
			Database: config.Database{
				DriverName:     "sqlite",
				DataSourceName: ":memory:?_journal=WAL&_timeout=5000&_fk=true",
			},
		},
	}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *QueueGetAllIntegrationTestSuite) TestGetQueue() {
	tests := []struct {
		name      string
		path      string
		setupMock func() *mocks.MockManager
		wantCode  int
		wantBody  string
	}{
		{
			name: "when get Ok",
			path: "/queue",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{
"items": [
  {
    "body": "test body",
    "created": "2024-09-10T12:00:00Z",
    "id": "message-id",
    "received": 5,
    "timeout": "2024-09-10T13:00:00Z",
    "updated": "2024-09-10T12:01:00Z"
  }
],
"total_items": 10
}`,
		},
		{
			name: "when GetAll errors",
			path: "/queue",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().GetAll(context.Background(), 10, 0).
					Return(nil, fmt.Errorf("GetAll error")).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0,"error":"GetAll error"}`,
		},
		{
			name: "when Count errors",
			path: "/queue",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().GetAll(gomock.Any(), 10, 0).Return(nil, nil)
				mock.EXPECT().Count(context.Background()).
					Return(0, fmt.Errorf("Count error")).AnyTimes()

				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"code":0,"error":"Count error"}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			a := api.New(suite.appConfig, suite.logger)
			queueGen.RegisterHandlers(a.Echo, queue.New(mock))

			params := url.Values{}
			params.Add("limit", "10")
			params.Add("offset", "0")

			// Parse URI and add query parameters to it
			u, err := url.Parse(tc.path)
			suite.Require().NoError(err)
			u.RawQuery = params.Encode() // Add the query parameters

			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)
			assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestQueueGetAllIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueueGetAllIntegrationTestSuite))
}
