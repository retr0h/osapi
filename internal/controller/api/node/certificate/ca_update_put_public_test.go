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

package certificate_test

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

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apicertificate "github.com/osapi-io/osapi/internal/controller/api/node/certificate"
	"github.com/osapi-io/osapi/internal/controller/api/node/certificate/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type CAUpdatePutPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicertificate.Certificate
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *CAUpdatePutPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *CAUpdatePutPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicertificate.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *CAUpdatePutPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *CAUpdatePutPublicTestSuite) TestPutNodeCertificateCa() {
	tests := []struct {
		name         string
		request      gen.PutNodeCertificateCaRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeCertificateCaResponseObject)
	}{
		{
			name: "success",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"my-ca","changed":true}`),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
				s.Equal("my-ca", *r.Results[0].Name)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  ptr.To(true),
							Data:     nil,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "body validation error empty object",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "not found error",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "nonexistent",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("", nil, errors.New("certificate not found"))
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "does not exist error",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "missing",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("", nil, errors.New("certificate does not exist"))
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Status:   job.StatusSkipped,
							Hostname: "server1",
							Error:    "certificate: operation not supported on this OS family",
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.CertificateCAMutationEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "server1",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				_, ok := resp.(gen.PutNodeCertificateCa500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "_all",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"my-ca","changed":true}`),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"my-ca","changed":true}`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with nil response data",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "_all",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  ptr.To(true),
							Data:     nil,
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast with failed host",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "_all",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.CertificateCAMutationEntryStatusFailed, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "unreachable")
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "_all",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "certificate: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				r, ok := resp.(gen.PutNodeCertificateCa200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.CertificateCAMutationEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PutNodeCertificateCaRequestObject{
				Hostname: "_all",
				Name:     "my-ca",
				Body: &gen.PutNodeCertificateCaJSONRequestBody{
					Object: "my-ca-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"certificate",
						job.OperationCertificateCAUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeCertificateCaResponseObject) {
				_, ok := resp.(gen.PutNodeCertificateCa500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PutNodeCertificateCa(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *CAUpdatePutPublicTestSuite) TestPutNodeCertificateCaValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/certificate/ca/my-ca",
			body: `{"object":"my-ca-object-v2"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "certificate", job.OperationCertificateCAUpdate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"my-ca","changed":true}`),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
			},
		},
		{
			name: "when missing object",
			path: "/api/node/server1/certificate/ca/my-ca",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Object")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/certificate/ca/my-ca",
			body: `{"object":"my-ca-object-v2"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			certificateHandler := apicertificate.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(certificateHandler, nil)

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

const rbacCertUpdateTestSigningKey = "test-signing-key-for-rbac-cert-update"

func (s *CAUpdatePutPublicTestSuite) TestPutNodeCertificateCaRBACHTTP() {
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
					rbacCertUpdateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"certificate:read"},
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
					rbacCertUpdateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "certificate", job.OperationCertificateCAUpdate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"my-ca","changed":true}`),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
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
							SigningKey: rbacCertUpdateTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apicertificate.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPut,
				"/api/node/server1/certificate/ca/my-ca",
				strings.NewReader(`{"object":"my-ca-object-v2"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestCAUpdatePutPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CAUpdatePutPublicTestSuite))
}
