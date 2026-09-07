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

	"k8s.io/utils/ptr"

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

type FileDeployPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *nodeFile.File
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *FileDeployPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *FileDeployPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = nodeFile.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileDeployPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileDeployPostPublicTestSuite) TestPostNodeFileDeploy() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.PostNodeFileDeployRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeFileDeployResponseObject)
	}{
		{
			name: "when success",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
					Mode:        ptr.To("0644"),
					Owner:       ptr.To("root"),
					Group:       ptr.To("root"),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileDeployExecute,
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
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
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
			name: "when success with template vars",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "app.conf.tmpl",
					Path:        "/etc/app/app.conf",
					ContentType: gen.Template,
					Vars: &map[string]interface{}{
						"port": float64(8080),
					},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileDeployExecute,
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
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error missing object_name",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "ObjectName")
			},
		},
		{
			name: "when validation error missing path",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Path")
			},
		},
		{
			name: "when validation error invalid content_type",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.FileDeployRequestContentType("invalid"),
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "ContentType")
			},
		},
		{
			name: "when broadcast success",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				changedFalse := false
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileDeployExecute,
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
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "when broadcast has errors",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileDeployExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"agent1": {Hostname: "agent1", Changed: &changedTrue},
							"agent2": {
								Status:   job.StatusFailed,
								Error:    "deploy failed",
								Hostname: "agent2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
				errCount := 0
				for _, res := range r.Results {
					if res.Error != nil {
						errCount++
						s.Equal("deploy failed", *res.Error)
					}
				}
				s.Equal(1, errCount)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileDeployExecute,
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
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.FileDeployResultStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileDeployExecute,
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
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.FileDeployResultStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast client error",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"file",
						job.OperationFileDeployExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				_, ok := resp.(gen.PostNodeFileDeploy500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job client error",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"file",
						job.OperationFileDeployExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				_, ok := resp.(gen.PostNodeFileDeploy500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeFileDeployRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeFileDeployJSONRequestBody{
					ObjectName:  "nginx.conf",
					Path:        "/etc/nginx/nginx.conf",
					ContentType: gen.Raw,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"file",
						job.OperationFileDeployExecute,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeFileDeployResponseObject) {
				r, ok := resp.(gen.PostNodeFileDeploy202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.FileDeployResultStatusSkipped, r.Results[0].Status)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeFileDeploy(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileDeployPostPublicTestSuite) TestPostNodeFileDeployValidationHTTP() {
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
			path: "/api/node/server1/file/deploy",
			body: `{"object_name":"nginx.conf","path":"/etc/nginx/nginx.conf","content_type":"raw"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "file", job.OperationFileDeployExecute, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "agent1")
				s.Contains(rec.Body.String(), "changed")
				s.Contains(rec.Body.String(), "results")
			},
		},
		{
			name: "when missing object_name",
			path: "/api/node/server1/file/deploy",
			body: `{"path":"/etc/nginx/nginx.conf","content_type":"raw"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "ObjectName")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when invalid content_type",
			path: "/api/node/server1/file/deploy",
			body: `{"object_name":"nginx.conf","path":"/etc/nginx/nginx.conf","content_type":"invalid"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "ContentType")
			},
		},
		{
			name: "when server error",
			path: "/api/node/server1/file/deploy",
			body: `{"object_name":"nginx.conf","path":"/etc/nginx/nginx.conf","content_type":"raw"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "file", job.OperationFileDeployExecute, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "error")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/file/deploy",
			body: `{"object_name":"nginx.conf","path":"/etc/nginx/nginx.conf","content_type":"raw"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
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

const rbacFileDeployTestSigningKey = "test-signing-key-for-file-deploy-rbac"

func (s *FileDeployPostPublicTestSuite) TestPostNodeFileDeployRBACHTTP() {
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
					rbacFileDeployTestSigningKey,
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
					rbacFileDeployTestSigningKey,
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
					Modify(gomock.Any(), "server1", "file", job.OperationFileDeployExecute, gomock.Any()).
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
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "changed")
				s.Contains(rec.Body.String(), "results")
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
							SigningKey: rbacFileDeployTestSigningKey,
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
				"/api/node/server1/file/deploy",
				strings.NewReader(
					`{"object_name":"nginx.conf","path":"/etc/nginx/nginx.conf","content_type":"raw"}`,
				),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileDeployPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileDeployPostPublicTestSuite))
}
