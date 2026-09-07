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
	dnsMocks "github.com/osapi-io/osapi/internal/provider/network/netplan/dns/mocks"
)

type ProcessorNetworkPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorNetworkPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorNetworkPublicTestSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *ProcessorNetworkPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorNetworkPublicTestSuite) TestProcessDNSDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() *dnsMocks.MockProvider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful DNS delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "dns.delete",
				Data:      json.RawMessage(`{"interface": "eth0"}`),
			},
			setupMock: func() *dnsMocks.MockProvider {
				m := dnsMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					DeleteNetplanConfig("eth0").
					Return(true, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Equal(true, response["success"])
				s.Equal(true, response["changed"])
				s.Equal("DNS configuration deleted", response["message"])
			},
		},
		{
			name: "DNS delete returns not changed",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "dns.delete",
				Data:      json.RawMessage(`{"interface": "eth0"}`),
			},
			setupMock: func() *dnsMocks.MockProvider {
				m := dnsMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					DeleteNetplanConfig("eth0").
					Return(false, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Equal(true, response["success"])
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "DNS delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "dns.delete",
				Data:      json.RawMessage(`{"interface": "eth0"}`),
			},
			setupMock: func() *dnsMocks.MockProvider {
				m := dnsMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					DeleteNetplanConfig("eth0").
					Return(false, errors.New("netplan remove failed"))
				return m
			},
			expectError: true,
			errorMsg:    "netplan remove failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dnsMock := tt.setupMock()

			processor := agent.NewNetworkProcessor(
				dnsMock, nil,
				nil,
				nil,
				slog.Default(),
			)
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

func TestProcessorNetworkPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorNetworkPublicTestSuite))
}
