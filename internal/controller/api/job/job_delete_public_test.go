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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apijob "github.com/osapi-io/osapi/internal/controller/api/job"
	"github.com/osapi-io/osapi/internal/controller/api/job/gen"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type JobDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apijob.Job
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *JobDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apijob.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *JobDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *JobDeletePublicTestSuite) TestDeleteJobByID() {
	tests := []struct {
		name         string
		request      gen.DeleteJobByIDRequestObject
		mockError    error
		expectMock   bool
		validateFunc func(resp gen.DeleteJobByIDResponseObject)
	}{
		{
			name: "success",
			request: gen.DeleteJobByIDRequestObject{
				Id: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			expectMock: true,
			validateFunc: func(resp gen.DeleteJobByIDResponseObject) {
				_, ok := resp.(gen.DeleteJobByID204Response)
				s.True(ok)
			},
		},
		{
			name: "not found",
			request: gen.DeleteJobByIDRequestObject{
				Id: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  fmt.Errorf("job not found: 660e8400-e29b-41d4-a716-446655440000"),
			expectMock: true,
			validateFunc: func(resp gen.DeleteJobByIDResponseObject) {
				_, ok := resp.(gen.DeleteJobByID404JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "job client error",
			request: gen.DeleteJobByIDRequestObject{
				Id: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
			},
			mockError:  assert.AnError,
			expectMock: true,
			validateFunc: func(resp gen.DeleteJobByIDResponseObject) {
				_, ok := resp.(gen.DeleteJobByID500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.expectMock {
				s.mockJobClient.EXPECT().
					DeleteJob(gomock.Any(), tt.request.Id.String()).
					Return(tt.mockError)
			}

			resp, err := s.handler.DeleteJobByID(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *JobDeletePublicTestSuite) TestDeleteJobByIDHTTP() {
	tests := []struct {
		name         string
		jobID        string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:  "when valid uuid",
			jobID: "550e8400-e29b-41d4-a716-446655440000",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					DeleteJob(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNoContent, rec.Code)
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
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			jobHandler := apijob.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(jobHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/job/"+tc.jobID,
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacJobDeleteTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *JobDeletePublicTestSuite) TestDeleteJobByIDRBACHTTP() {
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
					rbacJobDeleteTestSigningKey,
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
			name: "when valid token with job:write returns 204",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacJobDeleteTestSigningKey,
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
					DeleteJob(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNoContent, rec.Code)
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
							SigningKey: rbacJobDeleteTestSigningKey,
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
				http.MethodDelete,
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

func TestJobDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JobDeletePublicTestSuite))
}
