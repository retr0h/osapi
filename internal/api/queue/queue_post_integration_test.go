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
	"bytes"
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/queue"
	queueGen "github.com/retr0h/osapi/internal/api/queue/gen"
	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/queue/mocks"
)

type QueuePostIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appConfig config.Config
	logger    *slog.Logger
}

func (suite *QueuePostIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *QueuePostIntegrationTestSuite) TestPostQueue() {
	tests := []struct {
		name      string
		path      string
		body      string
		setupMock func() *mocks.MockManager
		wantCode  int
		wantBody  string
	}{
		{
			name: "when post Ok",
			path: "/queue",
			body: `{"body": "EhIKBzguOC44LjgKBzguOC40LjQ="}`,
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "when body is malformed",
			path: "/queue",
			body: `{"body": }`, // Malformed JSON
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"code=400, message=Syntax error: offset=10, error=invalid character '}' looking for beginning of value, internal=invalid character '}' looking for beginning of value"}`,
		},
		{
			name: "when body is empty",
			path: "/queue",
			body: `{"body": ""}`,
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantCode: http.StatusBadRequest,
			wantBody: `{"code":0,"error":"Body field is required and cannot be empty"}`,
		},
		{
			name: "when Put errors",
			path: "/queue",
			body: `{"body": "EhIKBzguOC44LjgKBzguOC40LjQ="}`,
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				expectedBody, err := base64.StdEncoding.DecodeString("EhIKBzguOC44LjgKBzguOC40LjQ=")
				suite.Require().NoError(err)

				mock.EXPECT().Put(context.Background(), expectedBody).
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

			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			assert.Equal(suite.T(), tc.wantCode, rec.Code)

			if tc.wantCode == http.StatusCreated {
				assert.Empty(suite.T(), rec.Body.String())
			} else {
				assert.JSONEq(suite.T(), tc.wantBody, rec.Body.String())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestQueuePostIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueuePostIntegrationTestSuite))
}
