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

package schedule_test

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
	apischedule "github.com/osapi-io/osapi/internal/controller/api/node/schedule"
	"github.com/osapi-io/osapi/internal/controller/api/node/schedule/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type CronGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apischedule.Schedule
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *CronGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *CronGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apischedule.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *CronGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *CronGetPublicTestSuite) TestGetNodeScheduleCronByName() {
	tests := []struct {
		name         string
		request      gen.GetNodeScheduleCronByNameRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeScheduleCronByNameResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal("backup", *r.Results[0].Name)
				s.Equal("0 2 * * *", *r.Results[0].Schedule)
				s.Equal("root", *r.Results[0].User)
				s.Equal("backup-script", *r.Results[0].Object)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "_all",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Data: json.RawMessage(
								`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with error entries",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "_all",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Status:   job.StatusCompleted,
							Data: json.RawMessage(
								`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "cron entry not found",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
				errCount := 0
				for _, res := range r.Results {
					if res.Error != nil {
						errCount++
						s.Equal("cron entry not found", *res.Error)
					}
				}
				s.Equal(1, errCount)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "_all",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "cron: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.CronEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "_all",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				_, ok := resp.(gen.GetNodeScheduleCronByName500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "",
				Name:     "backup",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found error",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "nonexistent",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "nonexistent"},
					).
					Return("", nil, errors.New("cron entry not found"))
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "not managed error",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "unmanaged",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "unmanaged"},
					).
					Return("", nil, errors.New("cron entry not managed"))
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not managed")
			},
		},
		{
			name: "does not exist error",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "missing",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "missing"},
					).
					Return("", nil, errors.New("cron entry does not exist"))
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "cron: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				r, ok := resp.(gen.GetNodeScheduleCronByName200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.CronEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeScheduleCronByNameRequestObject{
				Hostname: "server1",
				Name:     "backup",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronGet,
						map[string]string{"name": "backup"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeScheduleCronByNameResponseObject) {
				_, ok := resp.(gen.GetNodeScheduleCronByName500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeScheduleCronByName(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *CronGetPublicTestSuite) TestGetNodeScheduleCronByNameValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/schedule/cron/backup",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "schedule", job.OperationCronGet, map[string]string{"name": "backup"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
						),
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
			path: "/api/node/nonexistent/schedule/cron/backup",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "valid_target")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			scheduleHandler := apischedule.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(scheduleHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacCronGetTestSigningKey = "test-signing-key-for-rbac-cron-get"

func (s *CronGetPublicTestSuite) TestGetNodeScheduleCronByNameRBACHTTP() {
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
					rbacCronGetTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"cron:write"},
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
					rbacCronGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "schedule", job.OperationCronGet, map[string]string{"name": "backup"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`{"name":"backup","schedule":"0 2 * * *","user":"root","object":"backup-script"}`,
						),
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
							SigningKey: rbacCronGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apischedule.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/node/server1/schedule/cron/backup",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestCronGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CronGetPublicTestSuite))
}
