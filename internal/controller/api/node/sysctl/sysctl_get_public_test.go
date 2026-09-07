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

type SysctlGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apisysctl.Sysctl
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *SysctlGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *SysctlGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apisysctl.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *SysctlGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SysctlGetPublicTestSuite) TestGetNodeSysctlByKey() {
	tests := []struct {
		name         string
		request      gen.GetNodeSysctlByKeyRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeSysctlByKeyResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","value":"1"}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal("net.ipv4.ip_forward", *r.Results[0].Key)
				s.Equal("1", *r.Results[0].Value)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal("", *r.Results[0].Key)
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","value":"1"}`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","value":"0"}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with error entries",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Status:   job.StatusCompleted,
							Data: json.RawMessage(
								`{"key":"net.ipv4.ip_forward","value":"1"}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "sysctl entry not found",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
				errCount := 0
				for _, res := range r.Results {
					if res.Error != nil {
						errCount++
						s.Equal("sysctl entry not found", *res.Error)
					}
				}
				s.Equal(1, errCount)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "sysctl: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.SysctlEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "_all",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				_, ok := resp.(gen.GetNodeSysctlByKey500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found error",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "nonexistent.key",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "nonexistent.key"},
					).
					Return("", nil, errors.New("sysctl entry not found"))
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "not managed error",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "unmanaged.key",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "unmanaged.key"},
					).
					Return("", nil, errors.New("sysctl entry not managed"))
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not managed")
			},
		},
		{
			name: "does not exist error",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "missing.key",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "missing.key"},
					).
					Return("", nil, errors.New("sysctl entry does not exist"))
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
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
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				r, ok := resp.(gen.GetNodeSysctlByKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.SysctlEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeSysctlByKeyRequestObject{
				Hostname: "server1",
				Key:      "net.ipv4.ip_forward",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationSysctlGet,
						map[string]string{"key": "net.ipv4.ip_forward"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeSysctlByKeyResponseObject) {
				_, ok := resp.(gen.GetNodeSysctlByKey500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeSysctlByKey(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *SysctlGetPublicTestSuite) TestGetNodeSysctlByKeyValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/sysctl/net.ipv4.ip_forward",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationSysctlGet, map[string]string{"key": "net.ipv4.ip_forward"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","value":"1"}`,
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
			name: "when target agent not found",
			path: "/api/node/nonexistent/sysctl/net.ipv4.ip_forward",
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

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacSysctlGetTestSigningKey = "test-signing-key-for-rbac-sysctl-get"

func (s *SysctlGetPublicTestSuite) TestGetNodeSysctlByKeyRBACHTTP() {
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
					rbacSysctlGetTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"sysctl:write"},
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
					rbacSysctlGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationSysctlGet, map[string]string{"key": "net.ipv4.ip_forward"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"key":"net.ipv4.ip_forward","value":"1"}`,
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
							SigningKey: rbacSysctlGetTestSigningKey,
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
				http.MethodGet,
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

func TestSysctlGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlGetPublicTestSuite))
}
