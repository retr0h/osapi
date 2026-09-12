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

package api_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	agentAPI "github.com/osapi-io/osapi/internal/controller/api/agent"
	"github.com/osapi-io/osapi/internal/controller/api/health"
	jobAPI "github.com/osapi-io/osapi/internal/controller/api/job"
	nodeAPI "github.com/osapi-io/osapi/internal/controller/api/node"
	"github.com/osapi-io/osapi/internal/job/mocks"
)

type HandlerPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	server        *api.Server
}

func (s *HandlerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)

	appConfig := config.Config{
		Controller: config.Controller{
			API: config.APIServer{
				Security: config.ServerSecurity{
					SigningKey: "test-signing-key",
				},
			},
		},
	}

	s.server = api.New(appConfig, slog.Default())
}

func (s *HandlerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HandlerPublicTestSuite) TestRegisterHandlers() {
	tests := []struct {
		name         string
		validateFunc func(int, int)
	}{
		{
			name: "registers handlers with Echo",
			validateFunc: func(routesAfter int, routesBefore int) {
				s.Greater(routesAfter, routesBefore)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			signingKey := "test-signing-key"

			handlers := make([]func(e *echo.Echo), 0, 5)
			handlers = append(
				handlers,
				agentAPI.Handler(slog.Default(), s.mockJobClient, signingKey, nil, nil)...,
			)
			handlers = append(
				handlers,
				nodeAPI.Handler(slog.Default(), s.mockJobClient, signingKey, nil)...,
			)
			handlers = append(
				handlers,
				jobAPI.Handler(slog.Default(), s.mockJobClient, signingKey, nil)...,
			)
			checker := &health.NATSChecker{}
			handlers = append(
				handlers,
				health.Handler(
					slog.Default(),
					checker,
					time.Now(),
					"0.1.0",
					nil,
					nil,
					signingKey,
					nil,
				)...,
			)

			routesBefore := len(s.server.Echo.Router().Routes())
			s.server.RegisterHandlers(handlers)
			routesAfter := len(s.server.Echo.Router().Routes())
			tt.validateFunc(routesAfter, routesBefore)
		})
	}
}

func TestHandlerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandlerPublicTestSuite))
}
