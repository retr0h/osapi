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
	"bytes"
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

type PackageInstallPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apipackage.Package
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *PackageInstallPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *PackageInstallPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apipackage.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *PackageInstallPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PackageInstallPublicTestSuite) TestPostNodePackage() {
	changeBool := true
	tests := []struct {
		name         string
		request      gen.PostNodePackageRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodePackageResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodePackageRequestObject{
				Hostname: "server1",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageInstall,
						map[string]string{"name": "curl"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changeBool,
						Data:     json.RawMessage(`{"name":"curl","changed":true}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				r, ok := resp.(gen.PostNodePackage200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.PackageMutationResultStatusOk, r.Results[0].Status)
				s.Equal("curl", *r.Results[0].Name)
			},
		},
		{
			name: "validation error empty name",
			request: gen.PostNodePackageRequestObject{
				Hostname: "server1",
				Body:     &gen.PackageInstallRequest{Name: ""},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				r, ok := resp.(gen.PostNodePackage400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodePackageRequestObject{
				Hostname: "",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				r, ok := resp.(gen.PostNodePackage400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodePackageRequestObject{
				Hostname: "server1",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageInstall,
						map[string]string{"name": "curl"},
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
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				r, ok := resp.(gen.PostNodePackage200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.PackageMutationResultStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "job client error",
			request: gen.PostNodePackageRequestObject{
				Hostname: "server1",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPackageInstall,
						map[string]string{"name": "curl"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				_, ok := resp.(gen.PostNodePackage500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all includes failed and skipped agents",
			request: gen.PostNodePackageRequestObject{
				Hostname: "_all",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageInstall,
						map[string]string{"name": "curl"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Changed:  &changeBool,
								Data:     json.RawMessage(`{"name":"curl","changed":true}`),
							},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "apt: install failed",
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
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				r, ok := resp.(gen.PostNodePackage200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.PackageMutationResult)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Equal(gen.PackageMutationResultStatusOk, byHost["server1"].Status)
				s.Equal(gen.PackageMutationResultStatusFailed, byHost["server2"].Status)
				s.Equal(gen.PackageMutationResultStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.PostNodePackageRequestObject{
				Hostname: "_all",
				Body:     &gen.PackageInstallRequest{Name: "curl"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPackageInstall,
						map[string]string{"name": "curl"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodePackageResponseObject) {
				_, ok := resp.(gen.PostNodePackage500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodePackage(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *PackageInstallPublicTestSuite) TestPostNodePackageValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/package",
			body: `{"name":"curl"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				changeBool := true
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationPackageInstall,
						map[string]string{"name": "curl"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changeBool,
						Data:     json.RawMessage(`{"name":"curl","changed":true}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when empty name returns 400",
			path: "/api/node/server1/package",
			body: `{"name":""}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			packageHandler := apipackage.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(packageHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodPost, tc.path,
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacPackageInstallTestSigningKey = "test-signing-key-for-rbac-package-install"

func (s *PackageInstallPublicTestSuite) TestPostNodePackageRBACHTTP() {
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
					rbacPackageInstallTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"package:read"},
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
					rbacPackageInstallTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				changeBool := true
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationPackageInstall,
						map[string]string{"name": "curl"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changeBool,
						Data:     json.RawMessage(`{"name":"curl","changed":true}`),
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
							SigningKey: rbacPackageInstallTestSigningKey,
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
				http.MethodPost,
				"/api/node/server1/package",
				bytes.NewBufferString(`{"name":"curl"}`),
			)
			req.Header.Set("Content-Type", "application/json")
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

func TestPackageInstallPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageInstallPublicTestSuite))
}
