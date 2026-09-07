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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apijob "github.com/osapi-io/osapi/internal/controller/api/job"
	"github.com/osapi-io/osapi/internal/controller/api/job/gen"
	"github.com/osapi-io/osapi/internal/job/client"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type JobRetryPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apijob.Job
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *JobRetryPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *JobRetryPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apijob.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *JobRetryPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *JobRetryPublicTestSuite) TestRetryJobByID() {
	targetHostname := "_any"

	tests := []struct {
		name         string
		request      gen.RetryJobByIDRequestObject
		mockResult   *client.CreateJobResult
		mockError    error
		expectMock   bool
		validateFunc func(resp gen.RetryJobByIDResponseObject)
	}{
		{
			name: "success",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Body: &gen.RetryJobByIDJSONRequestBody{
					TargetHostname: &targetHostname,
				},
			},
			mockResult: &client.CreateJobResult{
				JobID:     "660e8400-e29b-41d4-a716-446655440000",
				Status:    "created",
				Revision:  1,
				Timestamp: "2026-02-19T00:00:00Z",
			},
			expectMock: true,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				r, ok := resp.(gen.RetryJobByID201JSONResponse)
				s.True(ok)
				s.Equal("660e8400-e29b-41d4-a716-446655440000", r.JobId.String())
				s.Equal("created", r.Status)
			},
		},
		{
			name: "success with nil body",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			mockResult: &client.CreateJobResult{
				JobID:     "770e8400-e29b-41d4-a716-446655440000",
				Status:    "created",
				Revision:  1,
				Timestamp: "2026-02-19T00:00:00Z",
			},
			expectMock: true,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				r, ok := resp.(gen.RetryJobByID201JSONResponse)
				s.True(ok)
				s.Equal("770e8400-e29b-41d4-a716-446655440000", r.JobId.String())
			},
		},
		{
			name: "validation error empty target hostname",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Body: func() *gen.RetryJobByIDJSONRequestBody {
					s := ""
					return &gen.RetryJobByIDJSONRequestBody{
						TargetHostname: &s,
					}
				}(),
			},
			expectMock: false,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				r, ok := resp.(gen.RetryJobByID400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "TargetHostname")
			},
		},
		{
			name: "not found",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  fmt.Errorf("job not found: 660e8400-e29b-41d4-a716-446655440000"),
			expectMock: true,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				_, ok := resp.(gen.RetryJobByID404JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "no operation data",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
			},
			mockError: fmt.Errorf(
				"job has no operation data: 770e8400-e29b-41d4-a716-446655440000",
			),
			expectMock: true,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				r, ok := resp.(gen.RetryJobByID400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "no operation data")
			},
		},
		{
			name: "job client error",
			request: gen.RetryJobByIDRequestObject{
				Id: uuid.MustParse("880e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  fmt.Errorf("failed to create retry job: connection refused"),
			expectMock: true,
			validateFunc: func(resp gen.RetryJobByIDResponseObject) {
				_, ok := resp.(gen.RetryJobByID500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.expectMock {
				var expectedTarget string
				if tt.request.Body != nil && tt.request.Body.TargetHostname != nil {
					expectedTarget = *tt.request.Body.TargetHostname
				}
				s.mockJobClient.EXPECT().
					RetryJob(gomock.Any(), tt.request.Id.String(), expectedTarget).
					Return(tt.mockResult, tt.mockError)
			}

			resp, err := s.handler.RetryJobByID(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *JobRetryPublicTestSuite) TestRetryJobByIDValidationHTTP() {
	tests := []struct {
		name         string
		jobID        string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:  "when valid request with target",
			jobID: "550e8400-e29b-41d4-a716-446655440000",
			body:  `{"target_hostname":"_any"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					RetryJob(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000", "_any").
					Return(&client.CreateJobResult{
						JobID:     "660e8400-e29b-41d4-a716-446655440000",
						Status:    "created",
						Revision:  1,
						Timestamp: "2026-02-19T00:00:00Z",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), `"job_id":"660e8400-e29b-41d4-a716-446655440000"`)
				s.Contains(rec.Body.String(), `"status":"created"`)
			},
		},
		{
			name:  "when valid request without body",
			jobID: "550e8400-e29b-41d4-a716-446655440000",
			body:  `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					RetryJob(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000", "").
					Return(&client.CreateJobResult{
						JobID:     "770e8400-e29b-41d4-a716-446655440000",
						Status:    "created",
						Revision:  1,
						Timestamp: "2026-02-19T00:00:00Z",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), `"job_id":"770e8400-e29b-41d4-a716-446655440000"`)
			},
		},
		{
			name:  "when invalid uuid",
			jobID: "not-a-uuid",
			body:  `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"message"`)
				s.Contains(rec.Body.String(), "Invalid format for parameter id")
			},
		},
		{
			name:  "when empty target hostname in body",
			jobID: "550e8400-e29b-41d4-a716-446655440000",
			body:  `{"target_hostname":""}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "TargetHostname")
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
				http.MethodPost,
				"/api/job/"+tc.jobID+"/retry",
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacJobRetryTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *JobRetryPublicTestSuite) TestRetryJobByIDRBACHTTP() {
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
					rbacJobRetryTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
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
			name: "when valid token with job:write returns 201",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacJobRetryTestSigningKey,
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
					RetryJob(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000", "_any").
					Return(&client.CreateJobResult{
						JobID:     "660e8400-e29b-41d4-a716-446655440000",
						Status:    "created",
						Revision:  1,
						Timestamp: "2026-02-19T00:00:00Z",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), `"job_id":"660e8400-e29b-41d4-a716-446655440000"`)
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
							SigningKey: rbacJobRetryTestSigningKey,
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
				http.MethodPost,
				"/api/job/550e8400-e29b-41d4-a716-446655440000/retry",
				strings.NewReader(`{"target_hostname":"_any"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestJobRetryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JobRetryPublicTestSuite))
}
