// Copyright (c) 2026 John Dewey

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

package schedule_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	apischedule "github.com/osapi-io/osapi/internal/controller/api/node/schedule"
	"github.com/osapi-io/osapi/internal/job/mocks"
)

type HandlerPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
}

func (s *HandlerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
}

func (s *HandlerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HandlerPublicTestSuite) TestHandler() {
	tests := []struct {
		name         string
		validateFunc func([]func(e *echo.Echo))
	}{
		{
			name: "returns handler functions",
			validateFunc: func(handlers []func(e *echo.Echo)) {
				s.NotEmpty(handlers)
			},
		},
		{
			name: "closure registers routes and middleware executes",
			validateFunc: func(handlers []func(e *echo.Echo)) {
				e := echo.New()
				for _, h := range handlers {
					h(e)
				}
				s.NotEmpty(e.Routes())

				req := httptest.NewRequest(http.MethodGet, "/api/node/hostname/schedule/cron", nil)
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handlers := apischedule.Handler(
				slog.Default(),
				s.mockJobClient,
				"test-signing-key",
				nil,
			)

			tt.validateFunc(handlers)
		})
	}
}

func TestHandlerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandlerPublicTestSuite))
}
