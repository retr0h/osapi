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

package node_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apinode "github.com/osapi-io/osapi/internal/controller/api/node"
	"github.com/osapi-io/osapi/internal/controller/api/node/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/disk"
	"github.com/osapi-io/osapi/internal/provider/node/host"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	"github.com/osapi-io/osapi/internal/provider/node/mem"
	"github.com/osapi-io/osapi/internal/validation"
)

type NodeStatusGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinode.Node
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NodeStatusGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NodeStatusGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinode.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NodeStatusGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NodeStatusGetPublicTestSuite) TestGetNodeStatus() {
	tests := []struct {
		name         string
		request      gen.GetNodeStatusRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeStatusResponseObject)
	}{
		{
			name:    "success",
			request: gen.GetNodeStatusRequestObject{Hostname: "_any"},
			setupMock: func() {
				statusResp := job.NodeStatusResponse{
					Hostname: "test-host",
					Uptime:   time.Hour,
				}
				data, _ := json.Marshal(statusResp)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "test-host",
						Data:     json.RawMessage(data),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name:      "validation error empty hostname",
			request:   gen.GetNodeStatusRequestObject{Hostname: ""},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name:    "job client error",
			request: gen.GetNodeStatusRequestObject{Hostname: "_any"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				_, ok := resp.(gen.GetNodeStatus500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job skipped",
			request: gen.GetNodeStatusRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.NodeStatusResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all success",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				status1 := job.NodeStatusResponse{Hostname: "server1", Uptime: time.Hour}
				status2 := job.NodeStatusResponse{Hostname: "server2", Uptime: 2 * time.Hour}
				data1, _ := json.Marshal(status1)
				data2, _ := json.Marshal(status2)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				for _, result := range r.Results {
					s.Require().NotNil(result.Changed)
					s.False(*result.Changed)
				}
			},
		},
		{
			name:    "broadcast all with errors",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				status1 := job.NodeStatusResponse{Hostname: "server1", Uptime: time.Hour}
				data1, _ := json.Marshal(status1)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "disk full",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, res := range r.Results {
					if res.Error != nil {
						foundError = true
						s.Equal("server2", res.Hostname)
						s.Equal("disk full", *res.Error)
					}
				}
				s.True(foundError)
			},
		},
		{
			name:    "broadcast result with empty hostname falls back to map key",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				// Response data has no hostname field so the fallback `status.Hostname = host` fires.
				emptyHostnameStatus := job.NodeStatusResponse{Uptime: time.Hour}
				data, _ := json.Marshal(emptyHostnameStatus)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data)},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
			},
		},
		{
			name:    "broadcast with skipped host",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.NodeStatusResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast with failed host",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				r, ok := resp.(gen.GetNodeStatus200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.NodeStatusResponseStatusFailed, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all error",
			request: gen.GetNodeStatusRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeStatusResponseObject) {
				_, ok := resp.(gen.GetNodeStatus500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeStatus(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NodeStatusGetPublicTestSuite) TestGetNodeStatusValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantBody     string
		wantContains []string
	}{
		{
			name: "when empty hostname returns 400",
			path: "/api/node/%20",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
		{
			name: "when get Ok",
			path: "/api/node/server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				statusResp := job.NodeStatusResponse{
					Hostname: "default-hostname",
					Uptime:   5 * time.Hour,
					OSInfo: &host.Result{
						Distribution: "Ubuntu",
						Version:      "24.04",
					},
					LoadAverages: &load.Result{
						Load1:  1,
						Load5:  0.5,
						Load15: 0.2,
					},
					MemoryStats: &mem.Result{
						Total:  8388608,
						Free:   4194304,
						Cached: 2097152,
					},
					DiskUsage: []disk.Result{
						{
							Name:  "/dev/disk1",
							Total: 500000000000,
							Used:  250000000000,
							Free:  250000000000,
						},
					},
				}
				data, _ := json.Marshal(statusResp)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "default-hostname",
							Data:     json.RawMessage(data),
						},
						nil,
					)
				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "results": [
    {
      "changed": false,
      "disks": [
        {
          "free": 250000000000,
          "name": "/dev/disk1",
          "total": 500000000000,
          "used": 250000000000
        }
      ],
      "hostname": "default-hostname",
      "load_average": {
        "1min": 1,
        "5min": 0.5,
        "15min": 0.2
      },
      "memory": {
        "free": 4194304,
        "total": 8388608,
        "used": 2097152
      },
      "os_info": {
        "distribution": "Ubuntu",
        "version": "24.04"
      },
      "status": "ok",
      "uptime": "0 days, 5 hours, 0 minutes"
    }
  ]
}
`,
		},
		{
			name: "when job client errors",
			path: "/api/node/server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when broadcast all",
			path: "/api/node/_all",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				status1 := job.NodeStatusResponse{Hostname: "server1", Uptime: time.Hour}
				status2 := job.NodeStatusResponse{Hostname: "server2", Uptime: 2 * time.Hour}
				data1, _ := json.Marshal(status1)
				data2, _ := json.Marshal(status2)
				mock.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"server1"`, `"server2"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			nodeHandler := apinode.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(nodeHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			if tc.wantBody != "" {
				s.JSONEq(tc.wantBody, rec.Body.String())
			}
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacStatusTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *NodeStatusGetPublicTestSuite) TestGetNodeStatusRBACHTTP() {
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
					rbacStatusTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
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
			name: "when valid token with node:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacStatusTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				statusResp := job.NodeStatusResponse{
					Hostname: "default-hostname",
					Uptime:   5 * time.Hour,
					OSInfo: &host.Result{
						Distribution: "Ubuntu",
						Version:      "24.04",
					},
					LoadAverages: &load.Result{
						Load1:  1,
						Load5:  0.5,
						Load15: 0.2,
					},
					MemoryStats: &mem.Result{
						Total:  8388608,
						Free:   4194304,
						Cached: 2097152,
					},
					DiskUsage: []disk.Result{
						{
							Name:  "/dev/disk1",
							Total: 500000000000,
							Used:  250000000000,
							Free:  250000000000,
						},
					},
				}
				data, _ := json.Marshal(statusResp)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeStatusGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "default-hostname",
							Data:     json.RawMessage(data),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "hostname")
				s.Contains(rec.Body.String(), "default-hostname")
				s.Contains(rec.Body.String(), "job_id")
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
							SigningKey: rbacStatusTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apinode.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func (s *NodeStatusGetPublicTestSuite) TestFormatDuration() {
	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{
			name:  "0 days, 0 hours, 0 minutes",
			input: time.Duration(0) * time.Second,
			want:  "0 days, 0 hours, 0 minutes",
		},
		{
			name:  "0 days, 0 hours, 1 minute",
			input: time.Duration(60) * time.Second,
			want:  "0 days, 0 hours, 1 minute",
		},
		{
			name:  "0 days, 1 hour, 0 minutes",
			input: time.Duration(3600) * time.Second,
			want:  "0 days, 1 hour, 0 minutes",
		},
		{
			name:  "1 day, 0 hours, 0 minutes",
			input: time.Duration(24*3600) * time.Second,
			want:  "1 day, 0 hours, 0 minutes",
		},
		{
			name:  "1 day, 1 hour, 1 minute",
			input: time.Duration(24*3600+3600+60) * time.Second,
			want:  "1 day, 1 hour, 1 minute",
		},
		{
			name:  "4 days, 1 hour, 25 minutes",
			input: time.Duration(int64(math.Trunc(350735.47))) * time.Second,
			want:  "4 days, 1 hour, 25 minutes",
		},
		{
			name:  "2 days, 2 hours, 2 minutes",
			input: time.Duration(2*24*3600+2*3600+2*60) * time.Second,
			want:  "2 days, 2 hours, 2 minutes",
		},
		{
			name:  "0 days, 0 hours, 59 minutes",
			input: time.Duration(59) * time.Minute,
			want:  "0 days, 0 hours, 59 minutes",
		},
		{
			name:  "0 days, 23 hours, 59 minutes",
			input: time.Duration(23*3600+59*60) * time.Second,
			want:  "0 days, 23 hours, 59 minutes",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := apinode.ExportFormatDuration(tc.input)
			s.Equal(tc.want, got)
		})
	}
}

func (s *NodeStatusGetPublicTestSuite) TestUint64ToInt() {
	tests := []struct {
		name  string
		input uint64
		want  int
	}{
		{
			name:  "when within bounds - small value",
			input: 123,
			want:  123,
		},
		{
			name:  "when within bounds - max int value",
			input: uint64(math.MaxInt),
			want:  math.MaxInt,
		},
		{
			name:  "when overflow value - just above max int",
			input: uint64(math.MaxInt) + 1,
			want:  math.MaxInt,
		},
		{
			name:  "when overflow value - large uint64",
			input: math.MaxUint64,
			want:  math.MaxInt,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result := apinode.ExportUint64ToInt(tc.input)
			s.Equal(tc.want, result)
		})
	}
}

func TestNodeStatusGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NodeStatusGetPublicTestSuite))
}
