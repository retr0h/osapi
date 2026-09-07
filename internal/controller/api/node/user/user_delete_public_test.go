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

type UserDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiuser.User
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *UserDeletePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *UserDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiuser.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *UserDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *UserDeletePublicTestSuite) TestDeleteNodeUser() {
	tests := []struct {
		name         string
		request      gen.DeleteNodeUserRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteNodeUserResponseObject)
	}{
		{
			name: "success",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1", Changed: ptr.To(true),
						Data: json.RawMessage(`{"name":"testuser","changed":true}`),
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				r, ok := resp.(gen.DeleteNodeUser200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "",
				Name:     "testuser",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				_, ok := resp.(gen.DeleteNodeUser400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "not found",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "server1",
				Name:     "missing",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserDelete, map[string]string{"name": "missing"}).
					Return("", nil, fmt.Errorf("user not found: missing"))
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				_, ok := resp.(gen.DeleteNodeUser404JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "job client error",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				_, ok := resp.(gen.DeleteNodeUser500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "server1",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				r, ok := resp.(gen.DeleteNodeUser200JSONResponse)
				s.True(ok)
				s.Equal(gen.UserMutationResultStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "broadcast target _all",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Changed:  ptr.To(true),
								Data:     json.RawMessage(`{"name":"testuser","changed":true}`),
							},
						}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				r, ok := resp.(gen.DeleteNodeUser200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Changed:  ptr.To(true),
								Data:     json.RawMessage(`{"name":"testuser","changed":true}`),
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
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				r, ok := resp.(gen.DeleteNodeUser200JSONResponse)
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
			request: gen.DeleteNodeUserRequestObject{
				Hostname: "_all",
				Name:     "testuser",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeUserResponseObject) {
				_, ok := resp.(gen.DeleteNodeUser500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.DeleteNodeUser(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

const rbacUserDeleteTestSigningKey = "test-signing-key-for-rbac-user-delete"

func (s *UserDeletePublicTestSuite) TestDeleteNodeUserRBACHTTP() {
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
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacUserDeleteTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "user", job.OperationUserDelete, map[string]string{"name": "testuser"}).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1", Changed: ptr.To(true),
						Data: json.RawMessage(`{"name":"testuser","changed":true}`),
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
						Security: config.ServerSecurity{SigningKey: rbacUserDeleteTestSigningKey},
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

			req := httptest.NewRequest(http.MethodDelete, "/api/node/server1/user/testuser", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)
			tc.validateFunc(rec.Code)
		})
	}
}

func TestUserDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UserDeletePublicTestSuite))
}
