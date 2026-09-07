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

package process_test

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
	processAPI "github.com/osapi-io/osapi/internal/controller/api/node/process"
	"github.com/osapi-io/osapi/internal/controller/api/node/process/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type ProcessGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *processAPI.Process
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ProcessGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ProcessGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = processAPI.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ProcessGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessGetPublicTestSuite) TestGetNodeProcessByPid() {
	tests := []struct {
		name         string
		request      gen.GetNodeProcessByPidRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeProcessByPidResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "server1",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"pid":1234,"name":"nginx","user":"www-data","state":"running","cpu_percent":2.5,"mem_percent":1.2,"mem_rss":12345678,"command":"nginx: master process","start_time":"2026-01-01T00:00:00Z"}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.ProcessGetEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Process)
				s.Equal(1234, *r.Results[0].Process.Pid)
				s.Equal("nginx", *r.Results[0].Process.Name)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "",
				Pid:      1234,
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found returns 404",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "server1",
				Pid:      99999,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 99999},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusFailed,
						Hostname: "server1",
						Error:    "process not found",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "server1",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				_, ok := resp.(gen.GetNodeProcessByPid500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "server1",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.ProcessGetEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "failed non-404 returns 500",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "server1",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusFailed,
						Hostname: "server1",
						Error:    "permission denied",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid500JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "permission denied")
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "_all",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Data: json.RawMessage(
								`{"pid":1234,"name":"nginx","user":"www-data","state":"running"}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "process not found",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "_all",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "unsupported",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				r, ok := resp.(gen.GetNodeProcessByPid200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
				s.Equal(gen.ProcessGetEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeProcessByPidRequestObject{
				Hostname: "_all",
				Pid:      1234,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessGet,
						map[string]int{"pid": 1234},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeProcessByPidResponseObject) {
				_, ok := resp.(gen.GetNodeProcessByPid500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeProcessByPid(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ProcessGetPublicTestSuite) TestGetNodeProcessByPidValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/process/1234",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationProcessGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"pid":1234,"name":"nginx","state":"running"}`,
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
			path: "/api/node/nonexistent/process/1234",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
				s.Contains(rec.Body.String(), "not found")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			processHandler := processAPI.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(processHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacProcessGetTestSigningKey = "test-signing-key-for-rbac-process-get"

func (s *ProcessGetPublicTestSuite) TestGetNodeProcessByPidRBACHTTP() {
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
					rbacProcessGetTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"docker:write"},
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
					rbacProcessGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationProcessGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"pid":1234,"name":"nginx","state":"running"}`,
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
							SigningKey: rbacProcessGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := processAPI.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/node/server1/process/1234",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestProcessGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessGetPublicTestSuite))
}
