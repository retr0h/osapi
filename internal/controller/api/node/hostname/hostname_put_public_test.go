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

package hostname_test

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
	apihostname "github.com/osapi-io/osapi/internal/controller/api/node/hostname"
	"github.com/osapi-io/osapi/internal/controller/api/node/hostname/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type HostnamePutPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apihostname.Hostname
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *HostnamePutPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *HostnamePutPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apihostname.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *HostnamePutPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HostnamePutPublicTestSuite) TestPutNodeHostname() {
	trueVal := true
	falseVal := false

	tests := []struct {
		name         string
		request      gen.PutNodeHostnameRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeHostnameResponseObject)
	}{
		{
			name: "when success single target",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &trueVal,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.HostnameUpdateResultItemStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Changed: &trueVal},
							"server2": {Hostname: "server2", Changed: &falseVal},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
			},
		},
		{
			name: "when validation error empty hostname body",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when bad target hostname",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job client error",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				_, ok := resp.(gen.PutNodeHostname500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "server1",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.HostnameUpdateResultItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name: "when broadcast all with errors",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Changed: &trueVal},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, item := range r.Results {
					if item.Error != nil {
						foundError = true
						s.Equal("server2", item.Hostname)
						s.Equal(gen.HostnameUpdateResultItemStatusFailed, item.Status)
					}
				}
				s.True(foundError)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusSkipped,
								Error:    "host: operation not supported on this OS family",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.HostnameUpdateResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				r, ok := resp.(gen.PutNodeHostname202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.HostnameUpdateResultItemStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all error",
			request: gen.PutNodeHostnameRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeHostnameJSONRequestBody{
					Hostname: "new-hostname",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationNodeHostnameUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeHostnameResponseObject) {
				_, ok := resp.(gen.PutNodeHostname500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PutNodeHostname(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *HostnamePutPublicTestSuite) TestPutNodeHostnameHTTP() {
	trueVal := true

	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/hostname",
			body: `{"hostname":"new-hostname"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationNodeHostnameUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), `"results"`)
				s.Contains(rec.Body.String(), `"agent1"`)
				s.Contains(rec.Body.String(), `"ok"`)
				s.Contains(rec.Body.String(), `"changed":true`)
			},
		},
		{
			name: "when missing hostname body returns 400",
			path: "/api/node/server1/hostname",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "Hostname")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when empty hostname path returns 400",
			path: "/api/node/%20/hostname",
			body: `{"hostname":"new-hostname"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			hostnameHandler := apihostname.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(hostnameHandler, nil)

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

			tc.validateFunc(rec)
		})
	}
}

const rbacHostnamePutTestSigningKey = "test-signing-key-for-hostname-put-rbac"

func (s *HostnamePutPublicTestSuite) TestPutNodeHostnameRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true

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
					rbacHostnamePutTestSigningKey,
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
			name: "when valid token with node:write returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacHostnamePutTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationNodeHostnameUpdate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &trueVal,
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), `"results"`)
				s.Contains(rec.Body.String(), `"changed":true`)
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
							SigningKey: rbacHostnamePutTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apihostname.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPut,
				"/api/node/server1/hostname",
				strings.NewReader(`{"hostname":"new-hostname"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestHostnamePutPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnamePutPublicTestSuite))
}
