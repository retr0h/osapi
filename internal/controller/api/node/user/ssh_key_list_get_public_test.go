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

package user_test

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
	apiuser "github.com/osapi-io/osapi/internal/controller/api/node/user"
	"github.com/osapi-io/osapi/internal/controller/api/node/user/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type SSHKeyListGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiuser.User
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *SSHKeyListGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *SSHKeyListGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiuser.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *SSHKeyListGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SSHKeyListGetPublicTestSuite) TestGetNodeUserSSHKey() {
	tests := []struct {
		name         string
		request      gen.GetNodeUserSSHKeyRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeUserSSHKeyResponseObject)
	}{
		{
			name: "success with keys",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"type":"ssh-ed25519","fingerprint":"SHA256:abc123","comment":"user@host"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Keys)
				s.Len(*r.Results[0].Keys, 1)
				s.Equal("ssh-ed25519", *(*r.Results[0].Keys)[0].Type)
				s.Equal("SHA256:abc123", *(*r.Results[0].Keys)[0].Fingerprint)
				s.Equal("user@host", *(*r.Results[0].Keys)[0].Comment)
			},
		},
		{
			name: "success with key without comment",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"type":"ssh-rsa","fingerprint":"SHA256:xyz789"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Keys)
				keys := *r.Results[0].Keys
				s.Require().Len(keys, 1)
				s.Nil(keys[0].Comment)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "ssh key: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.SSHKeyEntryStatusSkipped, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				_, ok := resp.(gen.GetNodeUserSSHKey500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "",
				Name:     "testuser",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "broadcast target _all",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"type":"ssh-ed25519","fingerprint":"SHA256:abc123","comment":"user@host"}]`,
								),
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 1)
			},
		},
		{
			name: "broadcast includes failed and skipped",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data: json.RawMessage(
									`[{"type":"ssh-ed25519","fingerprint":"SHA256:abc123"}]`,
								),
							},
							"server2": {
								Hostname: "server2",
								Status:   job.StatusFailed,
								Error:    "connection timeout",
							},
							"server3": {
								Hostname: "server3",
								Status:   job.StatusSkipped,
								Error:    "unsupported",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				r, ok := resp.(gen.GetNodeUserSSHKey200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.GetNodeUserSSHKeyRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeUserSSHKeyResponseObject) {
				_, ok := resp.(gen.GetNodeUserSSHKey500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeUserSSHKey(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *SSHKeyListGetPublicTestSuite) TestGetNodeUserSSHKeyValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/user/testuser/ssh-key",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
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
			path: "/api/node/nonexistent/user/testuser/ssh-key",
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

			userHandler := apiuser.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(userHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacSSHKeyListTestSigningKey = "test-signing-key-for-rbac-ssh-key-list"

func (s *SSHKeyListGetPublicTestSuite) TestGetNodeUserSSHKeyRBACHTTP() {
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
					rbacSSHKeyListTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"node:write"},
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
					rbacSSHKeyListTestSigningKey,
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
					Query(
						gomock.Any(),
						"server1",
						"user",
						job.OperationSSHKeyList,
						map[string]string{"username": "testuser"},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
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
							SigningKey: rbacSSHKeyListTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apiuser.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/node/server1/user/testuser/ssh-key",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestSSHKeyListGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SSHKeyListGetPublicTestSuite))
}
