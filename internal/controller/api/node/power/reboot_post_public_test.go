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

package power_test

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
	apipower "github.com/osapi-io/osapi/internal/controller/api/node/power"
	"github.com/osapi-io/osapi/internal/controller/api/node/power/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	powerProv "github.com/osapi-io/osapi/internal/provider/node/power"
	"github.com/osapi-io/osapi/internal/validation"
)

type RebootPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apipower.Power
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *RebootPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *RebootPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apipower.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *RebootPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *RebootPostPublicTestSuite) TestPostNodePowerReboot() {
	changedTrue := true

	tests := []struct {
		name         string
		request      gen.PostNodePowerRebootRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodePowerRebootResponseObject)
	}{
		{
			name: "success with body",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "server1",
				Body: &gen.PowerRequest{
					Delay:   intPtr(5),
					Message: strPtr("maintenance reboot"),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{
							Delay:   5,
							Message: "maintenance reboot",
						},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"action":"reboot","delay":5,"changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.Ok, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
				s.Require().NotNil(r.Results[0].Action)
				s.Equal("reboot", *r.Results[0].Action)
			},
		},
		{
			name: "success with nil body",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "server1",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"action":"reboot","delay":0,"changed":true}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.Ok, r.Results[0].Status)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "",
				Body:     nil,
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "server1",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "power: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "server1",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				_, ok := resp.(gen.PostNodePowerReboot500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "_all",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"action":"reboot","delay":0,"changed":true}`,
							),
						},
						"server2": {
							Hostname: "server2",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"action":"reboot","delay":0,"changed":true}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with failed and skipped agents",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "_all",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  &changedTrue,
							Data: json.RawMessage(
								`{"action":"reboot","delay":0,"changed":true}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server2",
						},
						"server3": {
							Status:   job.StatusSkipped,
							Error:    "power: operation not supported on this OS family",
							Hostname: "server3",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				r, ok := resp.(gen.PostNodePowerReboot200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 3)

				byHost := make(map[string]*gen.PowerResult)
				for i := range r.Results {
					byHost[r.Results[i].Hostname] = &r.Results[i]
				}

				s.Require().Contains(byHost, "server1")
				s.Equal(gen.Ok, byHost["server1"].Status)

				s.Require().Contains(byHost, "server2")
				s.Equal(gen.Failed, byHost["server2"].Status)
				s.Contains(*byHost["server2"].Error, "permission denied")

				s.Require().Contains(byHost, "server3")
				s.Equal(gen.Skipped, byHost["server3"].Status)
			},
		},
		{
			name: "broadcast job client error",
			request: gen.PostNodePowerRebootRequestObject{
				Hostname: "_all",
				Body:     nil,
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"node",
						job.OperationPowerReboot,
						powerProv.Opts{},
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodePowerRebootResponseObject) {
				_, ok := resp.(gen.PostNodePowerReboot500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodePowerReboot(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *RebootPostPublicTestSuite) TestPostNodePowerRebootValidationHTTP() {
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
			name: "when valid request with body",
			path: "/api/node/server1/power/reboot",
			body: `{"delay":5,"message":"maintenance"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationPowerReboot, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"action":"reboot","delay":5,"changed":true}`,
						),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/power/reboot",
			body: `{}`,
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

			powerHandler := apipower.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(powerHandler, nil)

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

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacPowerRebootTestSigningKey = "test-signing-key-for-rbac-power-reboot"

func (s *RebootPostPublicTestSuite) TestPostNodePowerRebootRBACHTTP() {
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
					rbacPowerRebootTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacPowerRebootTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationPowerReboot, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &changedTrue,
						Data: json.RawMessage(
							`{"action":"reboot","delay":0,"changed":true}`,
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
							SigningKey: rbacPowerRebootTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apipower.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/power/reboot",
				strings.NewReader(`{}`),
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

func TestRebootPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RebootPostPublicTestSuite))
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
