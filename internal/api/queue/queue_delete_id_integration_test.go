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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/queue/mocks"
)

type QueueDeleteIDIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *QueueDeleteIDIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *QueueDeleteIDIntegrationTestSuite) TestDeleteQueueID() {
	tests := []struct {
		name      string
		path      string
		setupMock func() *mocks.MockManager
		wantCode  int
		wantBody  string
	}{
		{
			name: "when delete Ok",
			path: "/queue/message-id",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusNoContent,
			wantBody: `{}`,
		},
		{
			name: "when DeleteByID finds no item (no rows affected)",
			path: "/queue/message-id",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().DeleteByID(context.Background(), "message-id").
					Return(errors.NewNotFoundError("no item found with ID message-id")).AnyTimes()

				return mock
			},
			wantCode: http.StatusNotFound,
			wantBody: `{"code":0,"error":"not found: no item found with ID message-id"}`,
		},
		{
			name: "when DeleteByID errors",
			path: "/queue/message-id",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().DeleteByID(context.Background(), "message-id").
					Return(assert.AnError).AnyTimes()

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
			queueGen.RegisterHandlers(a.Echo, queue.New(mock))

			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)

			if tc.wantCode == http.StatusNoContent {
				assert.Empty(suite.T(), rec.Body.String())
			} else {
				assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestQueueDeleteIDIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueueDeleteIDIntegrationTestSuite))
}
