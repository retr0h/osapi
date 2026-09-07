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

type PackageListGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apipackage.Package
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *PackageListGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *PackageListGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apipackage.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *PackageListGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PackageListGetPublicTestSuite) TestGetNodePackage() {
	tests := []struct {
		name         string
		request      gen.GetNodePackageRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodePackageResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodePackageRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"name":"curl","version":"7.68.0","status":"installed","description":"command line tool for transferring data","size":512}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.PackageEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Packages)
				s.Require().Len(*r.Results[0].Packages, 1)
				s.Equal("curl", *(*r.Results[0].Packages)[0].Name)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodePackageRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Packages)
				s.Empty(*r.Results[0].Packages)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodePackageRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodePackageRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageList,
						nil,
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
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.PackageEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.GetNodePackageRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				_, ok := resp.(gen.GetNodePackage500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all with multiple agents",
			request: gen.GetNodePackageRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"name":"curl","version":"7.68.0","status":"installed"}]`,
								),
							},
							"server2": {
								Hostname: "server2",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"name":"wget","version":"1.20","status":"installed"}]`,
								),
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast target _all includes failed and skipped agents",
			request: gen.GetNodePackageRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"name":"curl","version":"7.68.0","status":"installed"}]`,
								),
							},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "apt: operation not supported",
								Hostname: "server2",
							},
							"server3": {
								Status:   job.StatusSkipped,
								Error:    "apt: operation not supported on this OS family",
								Hostname: "server3",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				r, ok := resp.(gen.GetNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.PackageEntry)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.PackageEntryStatusOk, byHost["server1"].Status)
				s.Nil(byHost["server1"].Error)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.PackageEntryStatusFailed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "not supported")

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.PackageEntryStatusSkipped, byHost["server3"].Status)
				s.Contains(*byHost["server3"].Error, "not supported")
			},
		},
		{
			name: "broadcast job client error",
			request: gen.GetNodePackageRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageResponseObject) {
				_, ok := resp.(gen.GetNodePackage500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodePackage(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *PackageListGetPublicTestSuite) TestGetNodePackageValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/package",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationPackageList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/package",
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

const rbacPackageListTestSigningKey = "test-signing-key-for-rbac-package-list"

func (s *PackageListGetPublicTestSuite) TestGetNodePackageRBACHTTP() {
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
					rbacPackageListTestSigningKey,
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
					rbacPackageListTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationPackageList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
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
							SigningKey: rbacPackageListTestSigningKey,
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
				"/api/node/server1/package",
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

func TestPackageListGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageListGetPublicTestSuite))
}
