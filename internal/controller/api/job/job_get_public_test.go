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

package job_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apijob "github.com/osapi-io/osapi/internal/controller/api/job"
	"github.com/osapi-io/osapi/internal/controller/api/job/gen"
	jobtypes "github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type JobGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apijob.Job
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *JobGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apijob.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *JobGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *JobGetPublicTestSuite) TestGetJobByID() {
	tests := []struct {
		name         string
		request      gen.GetJobByIDRequestObject
		mockJob      *jobtypes.QueuedJob
		mockError    error
		expectMock   bool
		validateFunc func(resp gen.GetJobByIDResponseObject)
	}{
		{
			name: "success with basic fields",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:      "550e8400-e29b-41d4-a716-446655440000",
				Status:  "completed",
				Created: "2025-06-14T10:00:00Z",
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.Equal("550e8400-e29b-41d4-a716-446655440000", r.Id.String())
				s.Equal("completed", *r.Status)
				s.Nil(r.Operation)
				s.Nil(r.Error)
				s.Nil(r.Hostname)
				s.Nil(r.UpdatedAt)
				s.Nil(r.Result)
				s.Nil(r.Changed)
			},
		},
		{
			name: "success with all optional fields",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:        "660e8400-e29b-41d4-a716-446655440000",
				Status:    "failed",
				Created:   "2025-06-14T10:00:00Z",
				Operation: map[string]interface{}{"type": "node.hostname.get"},
				Error:     "disk full",
				Hostname:  "agent-1",
				UpdatedAt: "2025-06-14T10:05:00Z",
				Result:    json.RawMessage(`{"hostname":"server-01"}`),
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.Equal("660e8400-e29b-41d4-a716-446655440000", r.Id.String())
				s.Equal("failed", *r.Status)
				s.NotNil(r.Operation)
				s.Equal("node.hostname.get", (*r.Operation)["type"])
				s.NotNil(r.Error)
				s.Equal("disk full", *r.Error)
				s.NotNil(r.Hostname)
				s.Equal("agent-1", *r.Hostname)
				s.NotNil(r.UpdatedAt)
				s.Equal("2025-06-14T10:05:00Z", *r.UpdatedAt)
				s.NotNil(r.Result)
			},
		},
		{
			name: "changed field propagated from queued job",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("ee0e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: func() *jobtypes.QueuedJob {
				changed := true
				return &jobtypes.QueuedJob{
					ID:      "ee0e8400-e29b-41d4-a716-446655440000",
					Status:  "completed",
					Created: "2025-06-14T10:00:00Z",
					Result:  json.RawMessage(`{"path":"/etc/hosts"}`),
					Changed: &changed,
				}
			}(),
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.NotNil(r.Changed)
				s.True(*r.Changed)
			},
		},
		{
			name: "not found",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  fmt.Errorf("job not found: 770e8400-e29b-41d4-a716-446655440000"),
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				_, ok := resp.(gen.GetJobByID404JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "job client error",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  assert.AnError,
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				_, ok := resp.(gen.GetJobByID500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast job with multiple responses includes changed",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: func() *jobtypes.QueuedJob {
				changedTrue := true
				changedFalse := false
				return &jobtypes.QueuedJob{
					ID:      "990e8400-e29b-41d4-a716-446655440000",
					Status:  "completed",
					Created: "2025-06-14T10:00:00Z",
					Responses: map[string]jobtypes.Response{
						"server1": {
							Status:   "completed",
							Hostname: "server1",
							Data:     json.RawMessage(`{"hostname":"server1"}`),
							Changed:  &changedTrue,
						},
						"server2": {
							Status:   "completed",
							Hostname: "server2",
							Data:     json.RawMessage(`{"hostname":"server2"}`),
							Changed:  &changedFalse,
						},
					},
					AgentStates: map[string]jobtypes.AgentState{
						"server1": {
							Status:   "completed",
							Duration: "1.5s",
						},
						"server2": {
							Status:   "completed",
							Duration: "2.1s",
						},
					},
				}
			}(),
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.Equal("990e8400-e29b-41d4-a716-446655440000", r.Id.String())
				s.Equal("completed", *r.Status)
				s.NotNil(r.Responses)
				s.Len(*r.Responses, 2)
				// Verify per-agent Changed is propagated
				respMap := *r.Responses
				s.NotNil(respMap["server1"].Changed)
				s.True(*respMap["server1"].Changed)
				s.NotNil(respMap["server2"].Changed)
				s.False(*respMap["server2"].Changed)
				s.NotNil(r.AgentStates)
				s.Len(*r.AgentStates, 2)
			},
		},
		{
			name: "single response includes responses map",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:      "aa0e8400-e29b-41d4-a716-446655440000",
				Status:  "completed",
				Created: "2025-06-14T10:00:00Z",
				Responses: map[string]jobtypes.Response{
					"server1": {
						Status:   "completed",
						Hostname: "server1",
						Data:     json.RawMessage(`{"hostname":"server1"}`),
					},
				},
				Result: json.RawMessage(`{"hostname":"server1"}`),
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.NotNil(r.Responses)
				s.Len(*r.Responses, 1)
				s.NotNil(r.Result)
			},
		},
		{
			name: "agent states with errors",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:      "bb0e8400-e29b-41d4-a716-446655440000",
				Status:  "partial_failure",
				Created: "2025-06-14T10:00:00Z",
				Responses: map[string]jobtypes.Response{
					"server1": {
						Status:   "completed",
						Hostname: "server1",
						Data:     json.RawMessage(`{"hostname":"server1"}`),
					},
					"server2": {
						Status:   "failed",
						Hostname: "server2",
						Error:    "disk full",
					},
				},
				AgentStates: map[string]jobtypes.AgentState{
					"server1": {
						Status:   "completed",
						Duration: "1.5s",
					},
					"server2": {
						Status: "failed",
						Error:  "disk full",
					},
				},
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.NotNil(r.Responses)
				s.Len(*r.Responses, 2)
				s.NotNil(r.AgentStates)
				ws := *r.AgentStates
				s.NotNil(ws["server2"].Error)
				s.Equal("disk full", *ws["server2"].Error)
			},
		},
		{
			name: "success with timeline events",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("cc0e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:      "cc0e8400-e29b-41d4-a716-446655440000",
				Status:  "failed",
				Created: "2026-02-19T10:00:00Z",
				Timeline: []jobtypes.TimelineEvent{
					{
						Timestamp: time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC),
						Event:     "submitted",
						Hostname:  "_api",
						Message:   "Job submitted to queue",
					},
					{
						Timestamp: time.Date(2026, 2, 19, 10, 0, 1, 0, time.UTC),
						Event:     "acknowledged",
						Hostname:  "agent-1",
						Message:   "Job acknowledged by agent agent-1",
					},
					{
						Timestamp: time.Date(2026, 2, 19, 10, 0, 3, 0, time.UTC),
						Event:     "failed",
						Hostname:  "agent-1",
						Message:   "Job failed on agent-1",
						Error:     "timeout",
					},
					{
						Timestamp: time.Date(2026, 2, 19, 10, 5, 0, 0, time.UTC),
						Event:     "retried",
						Hostname:  "_api",
						Message:   "Job retried as dd0e8400-e29b-41d4-a716-446655440000",
					},
				},
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.Equal("cc0e8400-e29b-41d4-a716-446655440000", r.Id.String())
				s.Equal("failed", *r.Status)
				s.NotNil(r.Timeline)
				tl := *r.Timeline
				s.Len(tl, 4)

				s.Equal("submitted", *tl[0].Event)
				s.Equal("_api", *tl[0].Hostname)
				s.Equal("Job submitted to queue", *tl[0].Message)
				s.Nil(tl[0].Error)

				s.Equal("acknowledged", *tl[1].Event)
				s.Equal("agent-1", *tl[1].Hostname)

				s.Equal("failed", *tl[2].Event)
				s.NotNil(tl[2].Error)
				s.Equal("timeout", *tl[2].Error)

				s.Equal("retried", *tl[3].Event)
				s.Contains(*tl[3].Message, "dd0e8400")
			},
		},
		{
			name: "empty timeline omits field",
			request: gen.GetJobByIDRequestObject{
				Id: uuid.MustParse("dd0e8400-e29b-41d4-a716-446655440000"),
			},
			mockJob: &jobtypes.QueuedJob{
				ID:       "dd0e8400-e29b-41d4-a716-446655440000",
				Status:   "completed",
				Created:  "2026-02-19T10:00:00Z",
				Timeline: nil,
			},
			expectMock: true,
			validateFunc: func(resp gen.GetJobByIDResponseObject) {
				r, ok := resp.(gen.GetJobByID200JSONResponse)
				s.True(ok)
				s.Nil(r.Timeline)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.expectMock {
				s.mockJobClient.EXPECT().
					GetJobStatus(gomock.Any(), tt.request.Id.String()).
					Return(tt.mockJob, tt.mockError)
			}

			resp, err := s.handler.GetJobByID(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *JobGetPublicTestSuite) TestGetJobByIDHTTP() {
	tests := []struct {
		name         string
		jobID        string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:  "when valid uuid with changed field",
			jobID: "550e8400-e29b-41d4-a716-446655440000",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				changed := true
				mock.EXPECT().
					GetJobStatus(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(&jobtypes.QueuedJob{
						ID:      "550e8400-e29b-41d4-a716-446655440000",
						Status:  "completed",
						Created: "2026-02-19T00:00:00Z",
						Changed: &changed,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "id")
				s.Contains(rec.Body.String(), "550e8400-e29b-41d4-a716-446655440000")
				s.Contains(rec.Body.String(), "status")
				s.Contains(rec.Body.String(), "completed")
				s.Contains(rec.Body.String(), "changed")
			},
		},
		{
			name:  "when invalid uuid",
			jobID: "not-a-uuid",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "message")
				s.Contains(rec.Body.String(), "Invalid format for parameter id")
			},
		},
		{
			name:  "when job has timeline events",
			jobID: "660e8400-e29b-41d4-a716-446655440000",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetJobStatus(gomock.Any(), "660e8400-e29b-41d4-a716-446655440000").
					Return(&jobtypes.QueuedJob{
						ID:      "660e8400-e29b-41d4-a716-446655440000",
						Status:  "failed",
						Created: "2026-02-19T10:00:00Z",
						Timeline: []jobtypes.TimelineEvent{
							{
								Timestamp: time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC),
								Event:     "submitted",
								Hostname:  "_api",
								Message:   "Job submitted to queue",
							},
							{
								Timestamp: time.Date(2026, 2, 19, 10, 0, 3, 0, time.UTC),
								Event:     "failed",
								Hostname:  "agent-1",
								Message:   "Job failed on agent-1",
								Error:     "timeout",
							},
						},
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "timeline")
				s.Contains(rec.Body.String(), "submitted")
				s.Contains(rec.Body.String(), "failed")
				s.Contains(rec.Body.String(), "Job submitted to queue")
				s.Contains(rec.Body.String(), "timeout")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			jobHandler := apijob.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(jobHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/job/"+tc.jobID,
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacJobGetTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *JobGetPublicTestSuite) TestGetJobByIDRBACHTTP() {
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
					rbacJobGetTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
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
			name: "when valid token with job:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacJobGetTestSigningKey,
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
					GetJobStatus(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(&jobtypes.QueuedJob{
						ID:      "550e8400-e29b-41d4-a716-446655440000",
						Status:  "completed",
						Created: "2026-02-19T00:00:00Z",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "id")
				s.Contains(rec.Body.String(), "550e8400-e29b-41d4-a716-446655440000")
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
							SigningKey: rbacJobGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apijob.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/job/550e8400-e29b-41d4-a716-446655440000",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestJobGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JobGetPublicTestSuite))
}
