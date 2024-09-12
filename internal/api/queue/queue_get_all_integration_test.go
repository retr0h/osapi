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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/config"
	queueManager "github.com/retr0h/osapi/internal/queue"
)

type QueueGetAllIntegrationTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *QueueGetAllIntegrationTestSuite) SetupTest() {
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

func (suite *QueueGetAllIntegrationTestSuite) TestGetQueueAll() {
	tests := []struct {
		name      string
		uri       string
		setupMock func() *queueManager.Mock
		want      struct {
			code int
			body string
		}
	}{
		{
			name: "when http ok",
			uri:  "/queue",
			setupMock: func() *queueManager.Mock {
				mock := queueManager.NewDefaultMock()
				return mock
			},
			want: struct {
				code int
				body string
			}{
				code: http.StatusOK,
				body: `{
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
		},
		{
			name: "when GetAll errors",
			uri:  "/queue",
			setupMock: func() *queueManager.Mock {
				mock := queueManager.NewDefaultMock()
				mock.GetAllFunc = func() ([]queueManager.Item, error) {
					return nil, fmt.Errorf("GetAll error")
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
			name: "when Count errors",
			uri:  "/queue",
			setupMock: func() *queueManager.Mock {
				mock := queueManager.NewDefaultMock()
				mock.CountFunc = func() (int, error) {
					return 0, fmt.Errorf("Count error")
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
			uri:  "/notfound",
			setupMock: func() *queueManager.Mock {
				mock := queueManager.NewDefaultMock()
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
			queueGen.RegisterHandlers(a.Echo, queue.New(mock))

			// Create a new request to the system endpoint
			req := httptest.NewRequest(http.MethodGet, tc.uri, nil)
			rec := httptest.NewRecorder()

			// Serve the request
			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.want.code, rec.Code)

			if tc.want.code == http.StatusOK {
				assert.JSONEq(suite.T(), tc.want.body, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestQueueGetAllIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueueGetAllIntegrationTestSuite))
}
