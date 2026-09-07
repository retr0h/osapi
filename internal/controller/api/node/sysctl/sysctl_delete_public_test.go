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
	"errors"
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
	apisysctl "github.com/osapi-io/osapi/internal/controller/api/node/sysctl"
	"github.com/osapi-io/osapi/internal/controller/api/node/sysctl/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type SysctlDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apisysctl.Sysctl
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *SysctlDeletePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *SysctlDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apisysctl.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *SysctlDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SysctlDeletePublicTestSuite) TestDeleteNodeSysctl() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.DeleteNodeSysctlRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteNodeSysctlResponseObject)
	}{
		{
			name: "success",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl200JSONResponse)
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
			name: "validation error empty hostname",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found error",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "server1",
				Key:      "nonexistent.key",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "nonexistent.key"},
					).
					Return("", nil, errors.New("sysctl entry not found"))
			},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "does not exist error",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "server1",
				Key:      "missing.key",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "missing.key"},
					).
					Return("", nil, errors.New("sysctl entry does not exist"))
			},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
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
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl200JSONResponse)
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
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				_, ok := resp.(gen.DeleteNodeSysctl500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
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
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
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
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				r, ok := resp.(gen.DeleteNodeSysctl200JSONResponse)
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
			request: gen.DeleteNodeSysctlRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlDelete,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeSysctlResponseObject) {
				_, ok := resp.(gen.DeleteNodeSysctl500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.DeleteNodeSysctl(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *SysctlDeletePublicTestSuite) TestDeleteNodeSysctlValidationHTTP() {
	changedTrue := true

	tests := []struct {
		name         string
		path         string
		method       string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:   "when valid request",
			path:   "/api/node/server1/sysctl/net.ipv4.ip_forward",
			method: http.MethodDelete,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationSysctlDelete, map[string]string{"key": "net.ipv4.ip_forward"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
						),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
			},
		},
		{
			name:   "when target agent not found",
			path:   "/api/node/nonexistent/sysctl/net.ipv4.ip_forward",
			method: http.MethodDelete,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			sysctlHandler := apisysctl.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(sysctlHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacSysctlDeleteTestSigningKey = "test-signing-key-for-rbac-sysctl-delete"

func (s *SysctlDeletePublicTestSuite) TestDeleteNodeSysctlRBACHTTP() {
	changedTrue := true
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusUnauthorized, rec.Code)
				s.Contains(rec.Body.String(), "Bearer token required")
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacSysctlDeleteTestSigningKey,
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
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusForbidden, rec.Code)
				s.Contains(rec.Body.String(), "Insufficient permissions")
			},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacSysctlDeleteTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationSysctlDelete, map[string]string{"key": "net.ipv4.ip_forward"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","changed":true}`,
						),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacSysctlDeleteTestSigningKey,
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
				http.MethodDelete,
				"/api/node/server1/sysctl/net.ipv4.ip_forward",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestSysctlDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlDeletePublicTestSuite))
}
