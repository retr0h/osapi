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
	"strings"
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

type ProcessSignalPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *processAPI.Process
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ProcessSignalPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ProcessSignalPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = processAPI.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ProcessSignalPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessSignalPublicTestSuite) TestPostNodeProcessSignal() {
	tests := []struct {
		name         string
		request      gen.PostNodeProcessSignalRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeProcessSignalResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data: json.RawMessage(
							`{"pid":1234,"signal":"TERM","changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.Ok, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Pid)
				s.Equal(1234, *r.Results[0].Pid)
				s.Require().NotNil(r.Results[0].Signal)
				s.Equal("TERM", *r.Results[0].Signal)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error invalid signal",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: "INVALID",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Signal")
			},
		},
		{
			name: "not found returns 404",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      99999,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusFailed,
						Hostname: "server1",
						Error:    "process not found",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				_, ok := resp.(gen.PostNodeProcessSignal500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "failed non-404 returns 500",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "server1",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.KILL,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusFailed,
						Hostname: "server1",
						Error:    "permission denied",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal500JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "permission denied")
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "_all",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data: json.RawMessage(
								`{"pid":1234,"signal":"TERM","changed":true}`,
							),
						},
						"server2": {
							Status:   job.StatusSkipped,
							Error:    "unsupported",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed host",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "_all",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				r, ok := resp.(gen.PostNodeProcessSignal200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
				s.Equal(gen.Failed, r.Results[0].Status)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PostNodeProcessSignalRequestObject{
				Hostname: "_all",
				Pid:      1234,
				Body: &gen.PostNodeProcessSignalJSONRequestBody{
					Signal: gen.TERM,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationProcessSignal,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeProcessSignalResponseObject) {
				_, ok := resp.(gen.PostNodeProcessSignal500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeProcessSignal(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ProcessSignalPublicTestSuite) TestPostNodeProcessSignalValidationHTTP() {
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
			path: "/api/node/server1/process/1234/signal",
			body: `{"signal":"TERM"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationProcessSignal, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"pid":1234,"signal":"TERM","changed":true}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when invalid signal",
			path: "/api/node/server1/process/1234/signal",
			body: `{"signal":"INVALID"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/process/1234/signal",
			body: `{"signal":"TERM"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target", "not found"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			processHandler := processAPI.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(processHandler, nil)

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

const rbacProcessSignalTestSigningKey = "test-signing-key-for-rbac-process-signal"

func (s *ProcessSignalPublicTestSuite) TestPostNodeProcessSignalRBACHTTP() {
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
					rbacProcessSignalTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"process:read"},
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
					rbacProcessSignalTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationProcessSignal, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"pid":1234,"signal":"TERM","changed":true}`),
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
							SigningKey: rbacProcessSignalTestSigningKey,
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
				http.MethodPost,
				"/api/node/server1/process/1234/signal",
				strings.NewReader(`{"signal":"TERM"}`),
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

func TestProcessSignalPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessSignalPublicTestSuite))
}
