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

package ntp_test

import (
	"context"
	"encoding/json"
	"errors"
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
	apintp "github.com/osapi-io/osapi/internal/controller/api/node/ntp"
	"github.com/osapi-io/osapi/internal/controller/api/node/ntp/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type NtpDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apintp.Ntp
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NtpDeletePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NtpDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apintp.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NtpDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NtpDeletePublicTestSuite) TestDeleteNodeNtp() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.DeleteNodeNtpRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteNodeNtpResponseObject)
	}{
		{
			name: "success",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found error",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("", nil, errors.New("ntp configuration not found"))
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "does not exist error",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("", nil, errors.New("ntp configuration does not exist"))
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "ntp: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.NtpMutationResultStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				_, ok := resp.(gen.DeleteNodeNtp500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"changed":true}`,
							),
						},
						"server2": {
							Hostname: "server2",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"changed":true}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"changed":true}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server2",
						},
						"server3": {
							Status:   job.StatusSkipped,
							Error:    "ntp: operation not supported on this OS family",
							Hostname: "server3",
						},
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				r, ok := resp.(gen.DeleteNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.NtpMutationResult)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.NtpMutationResultStatusOk, byHost["server1"].Status)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.NtpMutationResultStatusFailed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "permission denied")

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.NtpMutationResultStatusSkipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.DeleteNodeNtpRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpDelete,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeNtpResponseObject) {
				_, ok := resp.(gen.DeleteNodeNtp500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.DeleteNodeNtp(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NtpDeletePublicTestSuite) TestDeleteNodeNtpValidationHTTP() {
	changedTrue := true

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/ntp",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationNtpDelete, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"changed":true}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/ntp",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			ntpHandler := apintp.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(ntpHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacNtpDeleteTestSigningKey = "test-signing-key-for-rbac-ntp-delete"

func (s *NtpDeletePublicTestSuite) TestDeleteNodeNtpRBACHTTP() {
	changedTrue := true
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacNtpDeleteTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"ntp:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacNtpDeleteTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationNtpDelete, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"changed":true}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacNtpDeleteTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apintp.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/node/server1/ntp",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

func TestNtpDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NtpDeletePublicTestSuite))
}
