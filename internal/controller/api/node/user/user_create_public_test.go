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
	"strings"
	"testing"

	"k8s.io/utils/ptr"

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

type UserCreatePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiuser.User
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *UserCreatePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *UserCreatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiuser.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *UserCreatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *UserCreatePublicTestSuite) TestPostNodeUser() {
	tests := []struct {
		name         string
		request      gen.PostNodeUserRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeUserResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeUserRequestObject{
				Hostname: "server1",
				Body: &gen.UserCreateRequest{
					Name: "newuser",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				r, ok := resp.(gen.PostNodeUser200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("newuser", *r.Results[0].Name)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty name",
			request: gen.PostNodeUserRequestObject{
				Hostname: "server1",
				Body: &gen.UserCreateRequest{
					Name: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				_, ok := resp.(gen.PostNodeUser400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeUserRequestObject{
				Hostname: "",
				Body: &gen.UserCreateRequest{
					Name: "newuser",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				_, ok := resp.(gen.PostNodeUser400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeUserRequestObject{
				Hostname: "server1",
				Body:     &gen.UserCreateRequest{Name: "newuser"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				r, ok := resp.(gen.PostNodeUser200JSONResponse)
				s.True(ok)
				s.Equal(gen.UserMutationResultStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeUserRequestObject{
				Hostname: "server1",
				Body:     &gen.UserCreateRequest{Name: "newuser"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				_, ok := resp.(gen.PostNodeUser500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all",
			request: gen.PostNodeUserRequestObject{
				Hostname: "_all",
				Body:     &gen.UserCreateRequest{Name: "newuser"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Changed:  ptr.To(true),
								Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
							},
						}, nil)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				r, ok := resp.(gen.PostNodeUser200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
			},
		},
		{
			name: "success with all optional fields",
			request: gen.PostNodeUserRequestObject{
				Hostname: "server1",
				Body: &gen.UserCreateRequest{
					Name:     "newuser",
					Uid:      ptr.To(1001),
					Gid:      ptr.To(1001),
					Home:     ptr.To("/home/newuser"),
					Shell:    ptr.To("/bin/zsh"),
					Groups:   &[]string{"sudo", "docker"},
					Password: ptr.To("secret123"),
					System:   ptr.To(false),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				r, ok := resp.(gen.PostNodeUser200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("newuser", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.PostNodeUserRequestObject{
				Hostname: "_all",
				Body:     &gen.UserCreateRequest{Name: "newuser"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Changed:  ptr.To(true),
								Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
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
						}, nil)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				r, ok := resp.(gen.PostNodeUser200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)

				byHost := make(map[string]gen.UserMutationResult)
				for _, res := range r.Results {
					byHost[res.Hostname] = res
				}

				s.Equal(gen.UserMutationResultStatusOk, byHost["server1"].Status)
				s.Equal(gen.UserMutationResultStatusFailed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "connection timeout")
				s.Equal(gen.UserMutationResultStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.PostNodeUserRequestObject{
				Hostname: "_all",
				Body:     &gen.UserCreateRequest{Name: "newuser"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserCreate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeUserResponseObject) {
				_, ok := resp.(gen.PostNodeUser500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.PostNodeUser(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *UserCreatePublicTestSuite) TestPostNodeUserValidationHTTP() {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "when valid request",
			body:     `{"name":"newuser"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "when missing name",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := jobmocks.NewMockJobClient(s.mockCtrl)
			if tc.wantCode == http.StatusOK {
				jobMock.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
					}, nil)
			}

			userHandler := apiuser.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(userHandler, nil)
			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/user",
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
		})
	}
}

const rbacUserCreateTestSigningKey = "test-signing-key-for-rbac-user-create"

func (s *UserCreatePublicTestSuite) TestPostNodeUserRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(int)
	}{
		{
			name:      "when no token returns 401",
			setupAuth: func(_ *http.Request) {},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusUnauthorized, got)
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacUserCreateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"user:read"},
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusForbidden, got)
			},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacUserCreateTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"name":"newuser","changed":true}`),
					}, nil)
				return mock
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusOK, got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{SigningKey: rbacUserCreateTestSigningKey},
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
				http.MethodPost,
				"/api/node/server1/user",
				strings.NewReader(`{"name":"newuser"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec.Code)
		})
	}
}

func TestUserCreatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UserCreatePublicTestSuite))
}
