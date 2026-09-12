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

package file_test

import (
	"context"
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
	nodeFile "github.com/osapi-io/osapi/internal/controller/api/node/file"
	"github.com/osapi-io/osapi/internal/controller/api/node/file/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type FileUndeployPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *nodeFile.File
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *FileUndeployPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *FileUndeployPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = nodeFile.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileUndeployPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileUndeployPostPublicTestSuite) TestPostNodeFileUndeploy() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.PostNodeFileUndeployRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeFileUndeployResponseObject)
	}{
		{
			name: "when undeploy succeeds",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &changedTrue,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Equal("550e8400-e29b-41d4-a716-446655440000", r.JobId.String())
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when invalid hostname",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation fails empty path",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Path")
			},
		},
		{
			name: "when broadcast succeeds",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				changedFalse := false
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {Hostname: "agent1", Changed: &changedTrue},
							"agent2": {Hostname: "agent2", Changed: &changedFalse},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "when broadcast has errors",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {Hostname: "agent1", Changed: &changedTrue},
							"agent2": {
								Status:   job.StatusFailed,
								Error:    "undeploy failed",
								Hostname: "agent2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
				errCount := 0
				for _, res := range r.Results {
					if res.Error != nil {
						errCount++
						s.Equal("undeploy failed", *res.Error)
					}
				}
				s.Equal(1, errCount)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {
								Status:   job.StatusSkipped,
								Error:    "host: operation not supported on this OS family",
								Hostname: "agent1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.FileUndeployResultStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "agent1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.FileUndeployResultStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast client error",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				_, ok := resp.(gen.PostNodeFileUndeploy500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when undeploy fails",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				_, ok := resp.(gen.PostNodeFileUndeploy500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeFileUndeployRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeFileUndeployJSONRequestBody{
					Path: "/etc/cron.d/backup",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"file",
						job.OperationFileUndeployExecute,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeFileUndeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileUndeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.FileUndeployResultStatusSkipped, r.Results[0].Status)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeFileUndeploy(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileUndeployPostPublicTestSuite) TestPostNodeFileUndeployValidationHTTP() {
	changedTrue := true

	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/file/undeploy",
			body: `{"path":"/etc/cron.d/backup"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "file", job.OperationFileUndeployExecute, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"agent1"`)
				s.Contains(rec.Body.String(), `"changed":true`)
				s.Contains(rec.Body.String(), `"results"`)
			},
		},
		{
			name: "when missing path",
			path: "/api/node/server1/file/undeploy",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "Path")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when server error",
			path: "/api/node/server1/file/undeploy",
			body: `{"path":"/etc/cron.d/backup"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "file", job.OperationFileUndeployExecute, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/file/undeploy",
			body: `{"path":"/etc/cron.d/backup"}`,
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

			fileHandler := nodeFile.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(fileHandler, nil)

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

			tc.validateFunc(rec)
		})
	}
}

const rbacFileUndeployTestSigningKey = "test-signing-key-for-file-undeploy-rbac"

func (s *FileUndeployPostPublicTestSuite) TestPostNodeFileUndeployRBACHTTP() {
	changedTrue := true
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
					rbacFileUndeployTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:read"},
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
			name: "when valid token with file:write returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacFileUndeployTestSigningKey,
					[]string{"admin"},
					"test-user",
					[]string{"file:write"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "file", job.OperationFileUndeployExecute, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &changedTrue,
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"changed":true`)
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
							SigningKey: rbacFileUndeployTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := nodeFile.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/file/undeploy",
				strings.NewReader(`{"path":"/etc/cron.d/backup"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileUndeployPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileUndeployPostPublicTestSuite))
}
