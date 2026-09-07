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

package sysctl_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apisysctl "github.com/osapi-io/osapi/internal/controller/api/node/sysctl"
	"github.com/osapi-io/osapi/internal/controller/api/node/sysctl/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	sysctlProv "github.com/osapi-io/osapi/internal/provider/node/sysctl"
	"github.com/osapi-io/osapi/internal/validation"
)

type SysctlCreatePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apisysctl.Sysctl
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *SysctlCreatePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *SysctlCreatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apisysctl.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *SysctlCreatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SysctlCreatePublicTestSuite) TestPostNodeSysctl() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.PostNodeSysctlRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeSysctlResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "server1",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal("net.ipv4.ip_forward", *r.Results[0].Key)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error missing key",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "server1",
				Body: &gen.SysctlCreateRequest{
					Key:   "",
					Value: "1",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Key")
			},
		},
		{
			name: "validation error missing value",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "server1",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Value")
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "server1",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "sysctl: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.SysctlMutationResultStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "server1",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				_, ok := resp.(gen.PostNodeSysctl500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "_all",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","changed":true}`,
							),
						},
						"server2": {
							Hostname: "server2",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","changed":true}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "_all",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","changed":true}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server2",
						},
						"server3": {
							Status:   job.StatusSkipped,
							Error:    "sysctl: operation not supported on this OS family",
							Hostname: "server3",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				r, ok := resp.(gen.PostNodeSysctl200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.SysctlMutationResult)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.SysctlMutationResultStatusOk, byHost["server1"].Status)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.SysctlMutationResultStatusFailed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "permission denied")

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.SysctlMutationResultStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.PostNodeSysctlRequestObject{
				Hostname: "_all",
				Body: &gen.SysctlCreateRequest{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlCreate,
						sysctlProv.Entry{
							Key:   "net.ipv4.ip_forward",
							Value: "1",
						},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeSysctlResponseObject) {
				_, ok := resp.(gen.PostNodeSysctl500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeSysctl(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *SysctlCreatePublicTestSuite) TestPostNodeSysctlValidationHTTP() {
	changedTrue := true

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
			path: "/api/node/server1/sysctl",
			body: `{"key":"net.ipv4.ip_forward","value":"1"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationSysctlCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when missing key returns 400",
			path: "/api/node/server1/sysctl",
			body: `{"value":"1"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Key"},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/sysctl",
			body: `{"key":"net.ipv4.ip_forward","value":"1"}`,
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

			sysctlHandler := apisysctl.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(sysctlHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodPost,
				tc.path,
				strings.NewReader(tc.body),
			)
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

const rbacSysctlCreateTestSigningKey = "test-signing-key-for-rbac-sysctl-create"

func (s *SysctlCreatePublicTestSuite) TestPostNodeSysctlRBACHTTP() {
	changedTrue := true
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
				// No auth header set
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
					rbacSysctlCreateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"sysctl:read"},
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
					rbacSysctlCreateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationSysctlCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
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
							SigningKey: rbacSysctlCreateTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apisysctl.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/sysctl",
				strings.NewReader(`{"key":"net.ipv4.ip_forward","value":"1"}`),
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

func TestSysctlCreatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlCreatePublicTestSuite))
}
