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

package packageapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apipackage "github.com/osapi-io/osapi/internal/controller/api/node/package"
	"github.com/osapi-io/osapi/internal/controller/api/node/package/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type PackageGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apipackage.Package
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *PackageGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *PackageGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apipackage.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *PackageGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PackageGetPublicTestSuite) TestGetNodePackageByName() {
	tests := []struct {
		name         string
		request      gen.GetNodePackageByNameRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodePackageByNameResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "server1",
				Name:     "curl",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "curl"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"curl","version":"7.68.0","status":"installed","description":"command line tool","size":1024}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Packages)
				s.Require().Len(*r.Results[0].Packages, 1)
				pkg := (*r.Results[0].Packages)[0]
				s.Equal("curl", *pkg.Name)
				s.Equal("7.68.0", *pkg.Version)
				s.Require().NotNil(pkg.Size)
				s.Equal(int64(1024), *pkg.Size)
			},
		},
		{
			name: "not found",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "server1",
				Name:     "nonexistent",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "nonexistent"},
					).
					Return("", nil, fmt.Errorf("package not found: nonexistent"))
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "not installed",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "server1",
				Name:     "removed-pkg",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "removed-pkg"},
					).
					Return("", nil, fmt.Errorf("package not installed: removed-pkg"))
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not installed")
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "",
				Name:     "curl",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "server1",
				Name:     "curl",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "curl"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "apt: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.PackageEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "server1",
				Name:     "curl",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "curl"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				_, ok := resp.(gen.GetNodePackageByName500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all with failed and skipped agents",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "_all",
				Name:     "curl",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "curl"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`{"name":"curl","version":"7.68.0","status":"installed","description":"curl","size":2048}`,
								),
							},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "package not found",
								Hostname: "server2",
							},
							"server3": {
								Status:   job.StatusSkipped,
								Error:    "apt: unsupported",
								Hostname: "server3",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				r, ok := resp.(gen.GetNodePackageByName200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.PackageEntry)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.PackageEntryStatusOk, byHost["server1"].Status)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.PackageEntryStatusFailed, byHost["server2"].Status)

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.PackageEntryStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.GetNodePackageByNameRequestObject{
				Hostname: "_all",
				Name:     "curl",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageGet,
						map[string]string{"name": "curl"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageByNameResponseObject) {
				_, ok := resp.(gen.GetNodePackageByName500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodePackageByName(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *PackageGetPublicTestSuite) TestGetNodePackageByNameValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/package/curl",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationPackageGet,
						map[string]string{"name": "curl"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"curl","version":"7.68.0","status":"installed"}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/package/curl",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			packageHandler := apipackage.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(packageHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacPackageGetTestSigningKey = "test-signing-key-for-rbac-package-get"

func (s *PackageGetPublicTestSuite) TestGetNodePackageByNameRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacPackageGetTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"package:write"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacPackageGetTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationPackageGet,
						map[string]string{"name": "curl"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"curl","version":"7.68.0","status":"installed"}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacPackageGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apipackage.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/node/server1/package/curl",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

func TestPackageGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageGetPublicTestSuite))
}
