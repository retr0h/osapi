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

type ProcessListPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *processAPI.Process
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ProcessListPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ProcessListPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = processAPI.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ProcessListPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessListPublicTestSuite) TestGetNodeProcess() {
	tests := []struct {
		name         string
		request      gen.GetNodeProcessRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeProcessResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"pid":1,"name":"init","user":"root","state":"running","cpu_percent":0.1,"mem_percent":0.5,"mem_rss":1024,"command":"/sbin/init","start_time":"2026-01-01T00:00:00Z"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.ProcessEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Processes)
				s.Len(*r.Results[0].Processes, 1)
				p := (*r.Results[0].Processes)[0]
				s.Equal(1, *p.Pid)
				s.Equal("init", *p.Name)
				s.Equal("root", *p.User)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Processes)
				s.Empty(*r.Results[0].Processes)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				_, ok := resp.(gen.GetNodeProcess500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.ProcessEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`[{"pid":1,"name":"init","user":"root","state":"running"}]`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Data: json.RawMessage(
								`[{"pid":2,"name":"systemd","user":"root","state":"running"}]`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Data:     json.RawMessage(`[]`),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "process: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				r, ok := resp.(gen.GetNodeProcess200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
				s.Equal(gen.ProcessEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeProcessRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeProcessResponseObject) {
				_, ok := resp.(gen.GetNodeProcess500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeProcess(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ProcessListPublicTestSuite) TestGetNodeProcessValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/process",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationProcessList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"results"`)
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/process",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
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

const rbacProcessListTestSigningKey = "test-signing-key-for-rbac-process-list"

func (s *ProcessListPublicTestSuite) TestGetNodeProcessRBACHTTP() {
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
					rbacProcessListTestSigningKey,
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
					rbacProcessListTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationProcessList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"results"`)
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
							SigningKey: rbacProcessListTestSigningKey,
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
				"/api/node/server1/process",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestProcessListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessListPublicTestSuite))
}
