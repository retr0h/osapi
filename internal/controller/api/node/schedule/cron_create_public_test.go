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
	apischedule "github.com/osapi-io/osapi/internal/controller/api/node/schedule"
	"github.com/osapi-io/osapi/internal/controller/api/node/schedule/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type CronCreatePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apischedule.Schedule
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *CronCreatePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *CronCreatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apischedule.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *CronCreatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *CronCreatePublicTestSuite) TestPostNodeScheduleCron() {
	tests := []struct {
		name         string
		request      gen.PostNodeScheduleCronRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeScheduleCronResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:        "backup",
					Schedule:    strPtr("0 2 * * *"),
					Object:      "backup-script",
					User:        strPtr("root"),
					ContentType: (*gen.CronCreateRequestContentType)(strPtr("template")),
					Vars:        &map[string]interface{}{"region": "us-east"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
				s.Equal("backup", *r.Results[0].Name)
			},
		},
		{
			name: "success with nil user",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("backup", *r.Results[0].Name)
			},
		},
		{
			name: "success with interval instead of schedule",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "daily-backup",
					Interval: intervalPtr(gen.CronCreateRequestIntervalDaily),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"daily-backup","changed":true}`),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("daily-backup", *r.Results[0].Name)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     nil,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "backup-script",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "backup-script",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "backup-script",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "cron: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.CronMutationResultStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "backup-script",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				_, ok := resp.(gen.PostNodeScheduleCron500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "body validation error empty name",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "validation error neither schedule nor interval",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:   "backup",
					Object: "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Schedule")
			},
		},
		{
			name: "validation error both schedule and interval",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Interval: intervalPtr(gen.CronCreateRequestIntervalDaily),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "validation error invalid cron schedule expression",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("not-a-cron"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "cron_schedule")
			},
		},
		{
			name: "body validation error empty command",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
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
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				r, ok := resp.(gen.PostNodeScheduleCron200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.CronMutationResultStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeScheduleCronRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeScheduleCronJSONRequestBody{
					Name:     "backup",
					Schedule: strPtr("0 2 * * *"),
					Object:   "/usr/bin/backup.sh",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"schedule",
						job.OperationCronCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeScheduleCronResponseObject) {
				_, ok := resp.(gen.PostNodeScheduleCron500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeScheduleCron(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *CronCreatePublicTestSuite) TestPostNodeScheduleCronValidationHTTP() {
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
			path: "/api/node/server1/schedule/cron",
			body: `{"name":"backup","schedule":"0 2 * * *","object":"backup-script"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "schedule", job.OperationCronCreate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						nil,
					)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when missing name",
			path: "/api/node/server1/schedule/cron",
			body: `{"schedule":"0 2 * * *","object":"backup-script"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Name", "required"},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/schedule/cron",
			body: `{"name":"backup","schedule":"0 2 * * *","object":"backup-script"}`,
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

			scheduleHandler := apischedule.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(scheduleHandler, nil)

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

const rbacCronCreateTestSigningKey = "test-signing-key-for-rbac-cron-create"

func (s *CronCreatePublicTestSuite) TestPostNodeScheduleCronRBACHTTP() {
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
					rbacCronCreateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"cron:read"},
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
					rbacCronCreateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "schedule", job.OperationCronCreate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"backup","changed":true}`),
						},
						nil,
					)
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
							SigningKey: rbacCronCreateTestSigningKey,
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
				http.MethodPost,
				"/api/node/server1/schedule/cron",
				strings.NewReader(
					`{"name":"backup","schedule":"0 2 * * *","object":"backup-script"}`,
				),
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

func TestCronCreatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CronCreatePublicTestSuite))
}
