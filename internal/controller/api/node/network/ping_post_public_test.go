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

package network_test

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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apinetwork "github.com/osapi-io/osapi/internal/controller/api/node/network"
	"github.com/osapi-io/osapi/internal/controller/api/node/network/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/ping"
	"github.com/osapi-io/osapi/internal/validation"
)

type NetworkPingPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkPingPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkPingPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkPingPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkPingPostPublicTestSuite) TestPostNodeNetworkPing() {
	tests := []struct {
		name         string
		request      gen.PostNodeNetworkPingRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeNetworkPingResponseObject)
	}{
		{
			name: "when success",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				pingResult := ping.Result{
					AvgRTT:          20 * time.Millisecond,
					MaxRTT:          25 * time.Millisecond,
					MinRTT:          15 * time.Millisecond,
					PacketLoss:      0,
					PacketsReceived: 3,
					PacketsSent:     3,
				}
				data, _ := json.Marshal(pingResult)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].PacketsSent)
				s.Equal(3, *r.Results[0].PacketsSent)
				s.Require().NotNil(r.Results[0].PacketsReceived)
				s.Equal(3, *r.Results[0].PacketsReceived)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when body validation error empty address",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when job client error",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				_, ok := resp.(gen.PostNodeNetworkPing500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.PingResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				ping1 := ping.Result{
					AvgRTT:          20 * time.Millisecond,
					PacketsSent:     3,
					PacketsReceived: 3,
				}
				ping2 := ping.Result{
					AvgRTT:          30 * time.Millisecond,
					PacketsSent:     3,
					PacketsReceived: 3,
				}
				data1, _ := json.Marshal(ping1)
				data2, _ := json.Marshal(ping2)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				for _, result := range r.Results {
					s.Require().NotNil(result.Changed)
					s.False(*result.Changed)
				}
			},
		},
		{
			name: "when broadcast all with errors",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				ping1 := ping.Result{
					AvgRTT:          20 * time.Millisecond,
					PacketsSent:     3,
					PacketsReceived: 3,
				}
				data1, _ := json.Marshal(ping1)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "host unreachable",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, h := range r.Results {
					if h.Error != nil {
						foundError = true
						s.Equal("server2", h.Hostname)
						s.Equal("host unreachable", *h.Error)
					}
				}
				s.True(foundError)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.PingResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkPing200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.PingResponseStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all error",
			request: gen.PostNodeNetworkPingRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeNetworkPingJSONRequestBody{
					Address: "8.8.8.8",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeNetworkPingResponseObject) {
				_, ok := resp.(gen.PostNodeNetworkPing500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeNetworkPing(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkPingPostPublicTestSuite) TestPostNetworkPingValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/ping",
			body: `{"address":"1.1.1.1"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				pingResult := ping.Result{
					PacketsSent:     3,
					PacketsReceived: 3,
					PacketLoss:      0,
					MinRTT:          10 * time.Millisecond,
					AvgRTT:          15 * time.Millisecond,
					MaxRTT:          20 * time.Millisecond,
				}
				data, _ := json.Marshal(pingResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "packets_sent")
				s.Contains(rec.Body.String(), "packets_received")
			},
		},
		{
			name: "when missing address",
			path: "/api/node/server1/network/ping",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Address")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when invalid address format",
			path: "/api/node/server1/network/ping",
			body: `{"address":"not-an-ip"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Address")
				s.Contains(rec.Body.String(), "ip_or_fact")
			},
		},
		{
			name: "when fact reference passes validation",
			path: "/api/node/server1/network/ping",
			body: `{"address":"@fact.custom.gateway"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				pingResult := ping.Result{
					PacketsSent:     3,
					PacketsReceived: 3,
					PacketLoss:      0,
					MinRTT:          10 * time.Millisecond,
					AvgRTT:          15 * time.Millisecond,
					MaxRTT:          20 * time.Millisecond,
				}
				data, _ := json.Marshal(pingResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "packets_sent")
			},
		},
		{
			name: "when partial fact reference rejected",
			path: "/api/node/server1/network/ping",
			body: `{"address":"@fact"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "ip_or_fact")
			},
		},
		{
			name: "when unknown fact key rejected",
			path: "/api/node/server1/network/ping",
			body: `{"address":"@fact.primary_interface"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "ip_or_fact")
			},
		},
		{
			name: "when broadcast all",
			path: "/api/node/_all/network/ping",
			body: `{"address":"1.1.1.1"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				pingResult := ping.Result{
					PacketsSent:     3,
					PacketsReceived: 3,
					PacketLoss:      0,
					MinRTT:          10 * time.Millisecond,
					AvgRTT:          15 * time.Millisecond,
					MaxRTT:          20 * time.Millisecond,
				}
				data, _ := json.Marshal(pingResult)
				mock.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data)},
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "packets_sent")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			networkHandler := apinetwork.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(networkHandler, nil)

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

const rbacPingTestSigningKey = "test-signing-key-for-ping-rbac"

func (s *NetworkPingPostPublicTestSuite) TestPostNetworkPingRBACHTTP() {
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
					rbacPingTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
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
			name: "when valid token with network:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacPingTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				pingResult := ping.Result{
					PacketsSent:     3,
					PacketsReceived: 3,
					PacketLoss:      0,
					MinRTT:          10 * time.Millisecond,
					AvgRTT:          15 * time.Millisecond,
					MaxRTT:          20 * time.Millisecond,
				}
				data, _ := json.Marshal(pingResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkPingDo, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "packets_sent")
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
							SigningKey: rbacPingTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apinetwork.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/network/ping",
				strings.NewReader(`{"address":"8.8.8.8"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func (s *NetworkPingPostPublicTestSuite) TestDurationToString() {
	dur := 20 * time.Millisecond

	tests := []struct {
		name         string
		d            *time.Duration
		validateFunc func(*string)
	}{
		{
			name: "when nil",
			d:    nil,
			validateFunc: func(got *string) {
				s.Equal((*string)(nil), got)
			},
		},
		{
			name: "when valid duration",
			d:    &dur,
			validateFunc: func(got *string) {
				s.Equal(func() *string { str := "20.00ms"; return &str }(), got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := apinetwork.ExportDurationToString(tc.d)
			tc.validateFunc(got)
		})
	}
}

func TestNetworkPingPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkPingPostPublicTestSuite))
}
