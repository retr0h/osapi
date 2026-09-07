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

type GroupListGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiuser.User
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *GroupListGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *GroupListGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiuser.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *GroupListGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *GroupListGetPublicTestSuite) TestGetNodeGroup() {
	tests := []struct {
		name         string
		request      gen.GetNodeGroupRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeGroupResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "group", job.OperationGroupList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"name":"root","gid":0},{"name":"sudo","gid":27,"members":["testuser"]}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				r, ok := resp.(gen.GetNodeGroup200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Groups)
				s.Len(*r.Results[0].Groups, 2)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				_, ok := resp.(gen.GetNodeGroup400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "group", job.OperationGroupList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				r, ok := resp.(gen.GetNodeGroup200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.GroupEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "group", job.OperationGroupList, nil).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				_, ok := resp.(gen.GetNodeGroup500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast target _all",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "group", job.OperationGroupList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data:     json.RawMessage(`[{"name":"root","gid":0}]`),
							},
							"server2": {
								Hostname: "server2",
								Status:   job.StatusSkipped,
								Error:    "unsupported",
							},
						}, nil)
			},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				r, ok := resp.(gen.GetNodeGroup200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "group", job.OperationGroupList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Hostname: "server1",
								Status:   job.StatusCompleted,
								Data:     json.RawMessage(`[{"name":"root","gid":0}]`),
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
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				r, ok := resp.(gen.GetNodeGroup200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 3)

				byHost := make(map[string]gen.GroupEntry)
				for _, res := range r.Results {
					byHost[res.Hostname] = res
				}

				s.Equal(gen.GroupEntryStatusOk, byHost["server1"].Status)
				s.Equal(gen.GroupEntryStatusFailed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "connection timeout")
				s.Equal(gen.GroupEntryStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.GetNodeGroupRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "group", job.OperationGroupList, nil).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeGroupResponseObject) {
				_, ok := resp.(gen.GetNodeGroup500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.GetNodeGroup(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

const rbacGroupListTestSigningKey = "test-signing-key-for-rbac-group-list"

func (s *GroupListGetPublicTestSuite) TestGetNodeGroupRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
	}{
		{
			name:      "when no token returns 401",
			setupAuth: func(_ *http.Request) {},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacGroupListTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"node:write"},
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode: http.StatusForbidden,
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacGroupListTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "group", job.OperationGroupList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1", Data: json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{SigningKey: rbacGroupListTestSigningKey},
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

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/group", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)
			s.Equal(tc.wantCode, rec.Code)
		})
	}
}

func TestGroupListGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GroupListGetPublicTestSuite))
}
