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

package log_test

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
	logAPI "github.com/osapi-io/osapi/internal/controller/api/node/log"
	"github.com/osapi-io/osapi/internal/controller/api/node/log/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type LogQueryPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *logAPI.Log
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *LogQueryPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *LogQueryPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = logAPI.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *LogQueryPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *LogQueryPublicTestSuite) TestGetNodeLog() {
	tests := []struct {
		name         string
		request      gen.GetNodeLogRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeLogResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"timestamp":"2026-01-01T00:00:00Z","unit":"sshd.service","priority":"info","message":"Started OpenSSH server","pid":1234,"hostname":"agent1"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.LogResultEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Entries)
				s.Len(*r.Results[0].Entries, 1)
				e := (*r.Results[0].Entries)[0]
				s.Equal("2026-01-01T00:00:00Z", *e.Timestamp)
				s.Equal("Started OpenSSH server", *e.Message)
			},
		},
		{
			name: "success with query params",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeLogParams{
					Lines:    intPtr(50),
					Since:    stringPtr("1 hour ago"),
					Priority: stringPtr("err"),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.LogResultEntryStatusOk, r.Results[0].Status)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeLogRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error invalid priority",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeLogParams{
					Priority: stringPtr("bogus"),
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "oneof")
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Entries)
				s.Empty(*r.Results[0].Entries)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				_, ok := resp.(gen.GetNodeLog500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeLogRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.LogResultEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeLogRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`[{"timestamp":"2026-01-01T00:00:00Z","priority":"info","message":"hello"}]`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Data:     json.RawMessage(`[]`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped hosts",
			request: gen.GetNodeLogRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
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
						"server3": {
							Status:   job.StatusSkipped,
							Error:    "log: operation not supported on this OS family",
							Hostname: "server3",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				r, ok := resp.(gen.GetNodeLog200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeLogRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationLogQuery,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeLogResponseObject) {
				_, ok := resp.(gen.GetNodeLog500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeLog(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *LogQueryPublicTestSuite) TestGetNodeLogHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/log",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationLogQuery, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
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
			path: "/api/node/nonexistent/log",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target", "not found"},
		},
		{
			name: "when invalid priority returns 400",
			path: "/api/node/server1/log?priority=bogus",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "oneof"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			logHandler := logAPI.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(logHandler, nil)

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

const rbacLogQueryTestSigningKey = "test-signing-key-for-rbac-log-query"

func (s *LogQueryPublicTestSuite) TestGetNodeLogRBACHTTP() {
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
					rbacLogQueryTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with log:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacLogQueryTestSigningKey,
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
					Query(gomock.Any(), "server1", "node", job.OperationLogQuery, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
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
							SigningKey: rbacLogQueryTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := logAPI.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/node/server1/log",
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

func intPtr(i int) *int          { return &i }
func stringPtr(s string) *string { return &s }

func TestLogQueryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LogQueryPublicTestSuite))
}
