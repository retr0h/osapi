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

type PackageUpdateGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apipackage.Package
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *PackageUpdateGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *PackageUpdateGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apipackage.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *PackageUpdateGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PackageUpdateGetPublicTestSuite) TestGetNodePackageUpdate() {
	tests := []struct {
		name         string
		request      gen.GetNodePackageUpdateRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodePackageUpdateResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"name":"curl","current_version":"7.68.0","new_version":"7.81.0"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				r, ok := resp.(gen.GetNodePackageUpdate200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.Ok, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Updates)
				s.Require().Len(*r.Results[0].Updates, 1)
				s.Equal("curl", *(*r.Results[0].Updates)[0].Name)
				s.Equal("7.68.0", *(*r.Results[0].Updates)[0].CurrentVersion)
				s.Equal("7.81.0", *(*r.Results[0].Updates)[0].NewVersion)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				r, ok := resp.(gen.GetNodePackageUpdate200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Updates)
				s.Empty(*r.Results[0].Updates)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				r, ok := resp.(gen.GetNodePackageUpdate400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "apt: unsupported",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				r, ok := resp.(gen.GetNodePackageUpdate200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				_, ok := resp.(gen.GetNodePackageUpdate500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all includes failed and skipped agents",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"name":"curl","current_version":"7.68.0","new_version":"7.81.0"}]`,
								),
							},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "apt: update check failed",
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
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				r, ok := resp.(gen.GetNodePackageUpdate200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.UpdateEntry)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.Ok, byHost["server1"].Status)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.Failed, byHost["server2"].Status)

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.Skipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.GetNodePackageUpdateRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageListUpdates,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodePackageUpdateResponseObject) {
				_, ok := resp.(gen.GetNodePackageUpdate500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodePackageUpdate(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *PackageUpdateGetPublicTestSuite) TestGetNodePackageUpdateValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/package/update",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationPackageListUpdates, nil).
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
			path: "/api/node/nonexistent/package/update",
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

const rbacPackageUpdateGetTestSigningKey = "test-signing-key-for-rbac-package-update-get"

func (s *PackageUpdateGetPublicTestSuite) TestGetNodePackageUpdateRBACHTTP() {
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
					rbacPackageUpdateGetTestSigningKey,
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
					rbacPackageUpdateGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationPackageListUpdates, nil).
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
							SigningKey: rbacPackageUpdateGetTestSigningKey,
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
				"/api/node/server1/package/update",
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

func TestPackageUpdateGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageUpdateGetPublicTestSuite))
}
