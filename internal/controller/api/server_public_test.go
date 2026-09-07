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
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/mock/gomock"

	auditmocks "github.com/osapi-io/osapi/internal/audit/mocks"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
)

type ServerPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ServerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ServerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ServerPublicTestSuite) TestNew() {
	tests := []struct {
		name      string
		appConfig config.Config
		opts      []api.Option
	}{
		{
			name: "creates server with default config",
			appConfig: config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: "test-key",
						},
					},
				},
			},
		},
		{
			name: "creates server with custom roles",
			appConfig: config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: "test-key",
							Roles: map[string]config.CustomRole{
								"ops": {
									Permissions: []string{"node:read", "health:read"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "creates server with audit store option",
			appConfig: config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: "test-key",
						},
					},
				},
			},
			opts: []api.Option{
				api.WithAuditStore(auditmocks.NewMockStore(s.mockCtrl)),
			},
		},
		{
			name: "creates server with meter provider option",
			appConfig: config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: "test-key",
						},
					},
				},
			},
			opts: []api.Option{
				api.WithMeterProvider(sdkmetric.NewMeterProvider()),
			},
		},
		{
			name: "creates server with CORS origins",
			appConfig: config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: "test-key",
							CORS: config.CORS{
								AllowOrigins: []string{
									"http://localhost:3000",
									"https://example.com",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			server := api.New(tt.appConfig, slog.Default(), tt.opts...)

			s.NotNil(server)
			s.NotNil(server.Echo)
		})
	}
}

func (s *ServerPublicTestSuite) TestStartAndStop() {
	s.Run("starts and stops gracefully", func() {
		appConfig := config.Config{
			Controller: config.Controller{
				API: config.APIServer{
					Port: 0,
					Security: config.ServerSecurity{
						SigningKey: "test-key",
					},
				},
			},
		}

		server := api.New(appConfig, slog.Default())
		server.Start()

		time.Sleep(50 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Stop(ctx)
	})
}

func (s *ServerPublicTestSuite) TestStartErrorPath() {
	s.Run("logs error when port already in use", func() {
		ln, err := net.Listen("tcp", ":0")
		s.Require().NoError(err)
		defer func() { _ = ln.Close() }()

		port := ln.Addr().(*net.TCPAddr).Port

		appConfig := config.Config{
			Controller: config.Controller{
				API: config.APIServer{
					Port: port,
					Security: config.ServerSecurity{
						SigningKey: "test-key",
					},
				},
			},
		}

		server := api.New(appConfig, slog.Default())
		server.Start()

		time.Sleep(100 * time.Millisecond)
	})
}

func (s *ServerPublicTestSuite) TestStopErrorPath() {
	s.Run("logs error when shutdown context expired with active connections", func() {
		ln, err := net.Listen("tcp", ":0")
		s.Require().NoError(err)
		port := ln.Addr().(*net.TCPAddr).Port
		_ = ln.Close()

		appConfig := config.Config{
			Controller: config.Controller{
				API: config.APIServer{
					Port: port,
					Security: config.ServerSecurity{
						SigningKey: "test-key",
					},
				},
			},
		}

		server := api.New(appConfig, slog.Default())

		server.Echo.GET("/slow", func(c echo.Context) error {
			time.Sleep(10 * time.Second)
			return c.String(http.StatusOK, "done")
		})

		server.Start()
		time.Sleep(50 * time.Millisecond)

		go http.Get(fmt.Sprintf("http://localhost:%d/slow", port)) //nolint:errcheck
		time.Sleep(50 * time.Millisecond)

		ctx, cancel := context.WithDeadline(
			context.Background(),
			time.Now().Add(-time.Second),
		)
		defer cancel()

		server.Stop(ctx)
	})
}

func TestServerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServerPublicTestSuite))
}
