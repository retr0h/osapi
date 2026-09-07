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
	"strings"
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
	ntpProv "github.com/osapi-io/osapi/internal/provider/node/ntp"
	"github.com/osapi-io/osapi/internal/validation"
)

type NtpUpdatePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apintp.Ntp
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NtpUpdatePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NtpUpdatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apintp.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NtpUpdatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NtpUpdatePublicTestSuite) TestPutNodeNtp() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.PutNodeNtpRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeNtpResponseObject)
	}{
		{
			name: "success",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "server1",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org", "1.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org", "1.pool.ntp.org"},
						},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty servers",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "server1",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Servers")
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "not found error",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "server1",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
					).
					Return("", nil, errors.New("ntp configuration not found"))
			},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "server1",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
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
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp200JSONResponse)
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
			request: gen.PutNodeNtpRequestObject{
				Hostname: "server1",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				_, ok := resp.(gen.PutNodeNtp500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "_all",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
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
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.PutNodeNtpRequestObject{
				Hostname: "_all",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
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
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				r, ok := resp.(gen.PutNodeNtp200JSONResponse)
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
			request: gen.PutNodeNtpRequestObject{
				Hostname: "_all",
				Body: &gen.NtpUpdateRequest{
					Servers: []string{"0.pool.ntp.org"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNtpUpdate,
						ntpProv.Config{
							Servers: []string{"0.pool.ntp.org"},
						},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNtpResponseObject) {
				_, ok := resp.(gen.PutNodeNtp500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PutNodeNtp(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NtpUpdatePublicTestSuite) TestPutNodeNtpValidationHTTP() {
	changedTrue := true

	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/ntp",
			body: `{"servers":["0.pool.ntp.org"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationNtpUpdate, gomock.Any()).
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
			name: "when missing servers returns 400",
			path: "/api/node/server1/ntp",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Servers"},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/ntp",
			body: `{"servers":["0.pool.ntp.org"]}`,
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

			req := httptest.NewRequest(
				http.MethodPut,
				tc.path,
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacNtpUpdateTestSigningKey = "test-signing-key-for-rbac-ntp-update"

func (s *NtpUpdatePublicTestSuite) TestPutNodeNtpRBACHTTP() {
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
					rbacNtpUpdateTestSigningKey,
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
					rbacNtpUpdateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationNtpUpdate, gomock.Any()).
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
							SigningKey: rbacNtpUpdateTestSigningKey,
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
				http.MethodPut,
				"/api/node/server1/ntp",
				strings.NewReader(`{"servers":["0.pool.ntp.org"]}`),
			)
			req.Header.Set("Content-Type", "application/json")
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

func TestNtpUpdatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NtpUpdatePublicTestSuite))
}
