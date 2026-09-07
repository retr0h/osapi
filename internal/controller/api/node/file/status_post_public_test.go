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
	nodeFile "github.com/osapi-io/osapi/internal/controller/api/node/file"
	"github.com/osapi-io/osapi/internal/controller/api/node/file/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	providerFile "github.com/osapi-io/osapi/internal/provider/file"
	"github.com/osapi-io/osapi/internal/validation"
)

type FileStatusPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *nodeFile.File
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *FileStatusPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *FileStatusPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = nodeFile.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileStatusPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func marshalStatusResult(
	r providerFile.StatusResult,
) json.RawMessage {
	b, _ := json.Marshal(r)
	return b
}

func (s *FileStatusPostPublicTestSuite) TestPostNodeFileStatus() {
	tests := []struct {
		name         string
		request      gen.PostNodeFileStatusRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeFileStatusResponseObject)
	}{
		{
			name: "when success with sha256",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data: marshalStatusResult(providerFile.StatusResult{
								Path:   "/etc/nginx/nginx.conf",
								Status: "in-sync",
								SHA256: "abc123def456",
							}),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Equal("550e8400-e29b-41d4-a716-446655440000", r.JobId.String())
				s.Require().Len(r.Results, 1)
				item := r.Results[0]
				s.Equal("agent1", item.Hostname)
				s.Require().NotNil(item.Path)
				s.Equal("/etc/nginx/nginx.conf", *item.Path)
				s.Require().NotNil(item.Status)
				s.Equal("in-sync", *item.Status)
				s.Require().NotNil(item.Sha256)
				s.Equal("abc123def456", *item.Sha256)
			},
		},
		{
			name: "when success missing file no sha256",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/missing.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data: marshalStatusResult(providerFile.StatusResult{
								Path:   "/etc/missing.conf",
								Status: "missing",
							}),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				item := r.Results[0]
				s.Require().NotNil(item.Status)
				s.Equal("missing", *item.Status)
				s.Nil(item.Sha256)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error missing path",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Path")
			},
		},
		{
			name: "when broadcast succeeds",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {
								Hostname: "agent1",
								Data: marshalStatusResult(providerFile.StatusResult{
									Path: "/etc/nginx/nginx.conf", Status: "in-sync", SHA256: "abc123",
								}),
							},
							"agent2": {
								Hostname: "agent2",
								Data: marshalStatusResult(providerFile.StatusResult{
									Path: "/etc/nginx/nginx.conf", Status: "missing",
								}),
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "when broadcast has errors",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {
								Hostname: "agent1",
								Data: marshalStatusResult(providerFile.StatusResult{
									Path: "/etc/nginx/nginx.conf", Status: "in-sync", SHA256: "abc123",
								}),
							},
							"agent2": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "agent2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
				errCount := 0
				for _, res := range r.Results {
					if res.Error != nil {
						errCount++
						s.Equal("permission denied", *res.Error)
					}
				}
				s.Equal(1, errCount)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileStatusGet,
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
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileStatusGet,
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
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast client error",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				_, ok := resp.(gen.PostNodeFileStatus500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job client error",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				_, ok := resp.(gen.PostNodeFileStatus500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeFileStatusRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeFileStatusJSONRequestBody{
					Path: "/etc/nginx/nginx.conf",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"file",
						job.OperationFileStatusGet,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeFileStatusResponseObject) {
				r, ok := resp.(gen.PostNodeFileStatus200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeFileStatus(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileStatusPostPublicTestSuite) TestPostNodeFileStatusValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/file/status",
			body: `{"path":"/etc/nginx/nginx.conf"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "file", job.OperationFileStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: marshalStatusResult(providerFile.StatusResult{
							Path:   "/etc/nginx/nginx.conf",
							Status: "in-sync",
							SHA256: "abc123",
						}),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"agent1"`)
				s.Contains(rec.Body.String(), `"in-sync"`)
				s.Contains(rec.Body.String(), `"sha256"`)
				s.Contains(rec.Body.String(), `"results"`)
			},
		},
		{
			name: "when missing path",
			path: "/api/node/server1/file/status",
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
			name: "when target agent not found",
			path: "/api/node/nonexistent/file/status",
			body: `{"path":"/etc/nginx/nginx.conf"}`,
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

const rbacFileStatusTestSigningKey = "test-signing-key-for-file-status-rbac"

func (s *FileStatusPostPublicTestSuite) TestPostNodeFileStatusRBACHTTP() {
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
					rbacFileStatusTestSigningKey,
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
			name: "when valid token with file:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacFileStatusTestSigningKey,
					[]string{"admin"},
					"test-user",
					[]string{"file:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "file", job.OperationFileStatusGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data: marshalStatusResult(providerFile.StatusResult{
								Path:   "/etc/nginx/nginx.conf",
								Status: "in-sync",
								SHA256: "abc123",
							}),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"in-sync"`)
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
							SigningKey: rbacFileStatusTestSigningKey,
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
				"/api/node/server1/file/status",
				strings.NewReader(`{"path":"/etc/nginx/nginx.conf"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileStatusPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileStatusPostPublicTestSuite))
}
