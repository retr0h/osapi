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

package agent_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/node/certificate"
	certMocks "github.com/osapi-io/osapi/internal/provider/node/certificate/mocks"
)

type ProcessorCertificatePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorCertificatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorCertificatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorCertificatePublicTestSuite) TestProcessCertificateOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() certificate.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "certificate provider not available",
		},
		{
			name: "dispatches to ca operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]certificate.Entry{}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []certificate.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Empty(entries)
			},
		},
		{
			name: "unsupported certificate operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "unknown.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported certificate operation: unknown.list",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var certProvider certificate.Provider
			if tt.setupMock != nil {
				certProvider = tt.setupMock()
			}

			processor := agent.NewCertificateProcessor(certProvider, slog.Default())
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func (s *ProcessorCertificatePublicTestSuite) TestProcessCertificateCAOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() certificate.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "invalid CA operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid certificate CA operation: ca",
		},
		{
			name: "successful CA list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]certificate.Entry{
					{
						Name:   "my-ca",
						Source: "managed",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []certificate.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("my-ca", entries[0].Name)
				s.Equal("managed", entries[0].Source)
			},
		},
		{
			name: "CA list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			expectError: true,
			errorMsg:    "permission denied",
		},
		{
			name: "successful CA create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.create",
				Data:      json.RawMessage(`{"name":"my-ca","object":"ca-cert-obj"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry certificate.Entry) (*certificate.CreateResult, error) {
						s.Equal("my-ca", entry.Name)
						return &certificate.CreateResult{
							Name:    "my-ca",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validate: func(result json.RawMessage) {
				var r certificate.CreateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-ca", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "CA create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal certificate CA create data",
		},
		{
			name: "CA create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.create",
				Data:      json.RawMessage(`{"name":"dup","object":"ca-cert-obj"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("already exists"))
				return m
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name: "successful CA update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.update",
				Data:      json.RawMessage(`{"name":"my-ca","object":"ca-cert-obj-v2"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry certificate.Entry) (*certificate.UpdateResult, error) {
						s.Equal("my-ca", entry.Name)
						return &certificate.UpdateResult{
							Name:    "my-ca",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validate: func(result json.RawMessage) {
				var r certificate.UpdateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-ca", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "CA update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal certificate CA update data",
		},
		{
			name: "CA update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.update",
				Data:      json.RawMessage(`{"name":"missing","object":"ca-cert-obj"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "successful CA delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.delete",
				Data:      json.RawMessage(`{"name":"my-ca"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "my-ca").Return(&certificate.DeleteResult{
					Name:    "my-ca",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r certificate.DeleteResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-ca", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "CA delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal certificate CA delete data",
		},
		{
			name: "CA delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "certificate",
				Operation: "ca.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() certificate.Provider {
				m := certMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "unsupported CA sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "certificate",
				Operation: "ca.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() certificate.Provider {
				return certMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported certificate CA operation: ca.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewCertificateProcessor(tt.setupMock(), slog.Default())
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func TestProcessorCertificatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorCertificatePublicTestSuite))
}
