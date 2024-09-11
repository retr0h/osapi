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
	// "fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/config"
	queueManager "github.com/retr0h/osapi/internal/queue"
)

type QueueGetIntegrationTestSuite struct {
	suite.Suite

	appFs     afero.Fs
	appConfig config.Config
	logger    *slog.Logger
	created   time.Time
	updated   time.Time
	timeout   time.Time
	received  int
}

func (suite *QueueGetIntegrationTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.appConfig = config.Config{
		Queue: config.Queue{
			Database: config.Database{
				DriverName:     "sqlite3",
				DataSourceName: ":memory:?_journal=WAL&_timeout=5000&_fk=true",
			},
		},
	}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	suite.created = time.Now()
	suite.updated = suite.created.Add(time.Minute)
	suite.timeout = suite.created.Add(time.Hour)
	suite.received = 10
}

func (suite *QueueGetIntegrationTestSuite) TestGetSystemStatus() {
	tests := []struct {
		name string
		uri  string
		// setupMock func() *systemProvider.MockSystem
		want struct {
			code int
			body string
		}
	}{
		// {
		// 	name: "when http ok",
		// 	uri:  "/queue",
		// 	// setupMock: func() *systemProvider.MockSystem {
		// 	// 	mock := systemProvider.NewDefaultMockSystem()
		// 	// 	return mock
		// 	// },
		// 	want: struct {
		// 		code int
		// 		body string
		// 	}{
		// 		code: http.StatusOK,
		// 		body: fmt.Sprintf(`{
		// "body":"example body",
		// "created":"%s",
		// "id":"foo",
		// "received":%d,
		// "timeout":"%s",
		// "updated":"%s"
		// }`, suite.created, suite.received, suite.timeout, suite.updated),
		// 	},
		// },
		// {
		// 	name: "when GetHostname errors",
		// 	uri:  "/system/status",
		// 	setupMock: func() *systemProvider.MockSystem {
		// 		mock := systemProvider.NewDefaultMockSystem()
		// 		mock.GetHostnameFunc = func() (string, error) {
		// 			return "", fmt.Errorf("GetHostname error")
		// 		}
		// 		return mock
		// 	},
		// 	want: struct {
		// 		code int
		// 		body string
		// 	}{
		// 		code: http.StatusInternalServerError,
		// 		body: `{}`,
		// 	},
		// },
		{
			name: "when not found",
			uri:  "/system/notfound",
			// setupMock: func() *systemProvider.MockSystem {
			// 	mock := systemProvider.NewDefaultMockSystem()
			// 	return mock
			// },
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
			// mock := tc.setupMock()
			a := api.New(suite.appFs, suite.appConfig, suite.logger)
			db, _ := queueManager.OpenDB(suite.appConfig)
			var qm queueManager.Manager = queueManager.New(suite.logger, suite.appConfig, db)
			_ = qm.SetupSchema(context.Background())
			_ = qm.SetupQueue()

			queueGen.RegisterHandlers(a, queue.New(qm))

			// Create a new request to the system endpoint
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
func TestQueueGetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueueGetIntegrationTestSuite))
}
