// Copyright (c) 2025 John Dewey

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

package job_test

import (
	"errors"
	"testing"

	hostpkg "github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	hostnamemocks "github.com/osapi-io/osapi/internal/job/mocks/hostname"
)

type HostnamePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *HostnamePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *HostnamePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HostnamePublicTestSuite) TestGetAgentHostnameProviderError() {
	tests := []struct {
		name         string
		validateFunc func(string, error)
	}{
		{
			name: "falls back to unknown when provider errors",
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("unknown", hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			mockProvider := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
			mockProvider.EXPECT().Hostname().Return("", errors.New("provider error"))

			job.SetDefaultHostnameProvider(mockProvider)
			defer job.ResetDefaultHostnameProvider()

			tt.validateFunc(job.GetAgentHostname(""))
		})
	}
}

func (s *HostnamePublicTestSuite) TestGetAgentHostname() {
	tests := []struct {
		name               string
		configuredHostname string
		expectError        bool
		expectedResult     string
		validateFunc       func(string)
	}{
		{
			name:               "configured hostname takes precedence",
			configuredHostname: "configured-agent",
			expectError:        false,
			expectedResult:     "configured-agent",
			validateFunc: func(hostname string) {
				s.Equal("configured-agent", hostname)
			},
		},
		{
			name:               "empty configured hostname falls back to system",
			configuredHostname: "",
			expectError:        false,
			validateFunc: func(hostname string) {
				s.NotEmpty(hostname, "System hostname should not be empty")
				s.NotEqual(
					"unknown",
					hostname,
					"Should get actual system hostname, not unknown fallback",
				)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			hostname, err := job.GetAgentHostname(tt.configuredHostname)

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(hostname)
				}
				if tt.expectedResult != "" {
					s.Equal(tt.expectedResult, hostname)
				}
			}
		})
	}
}

func (s *HostnamePublicTestSuite) TestGetLocalHostname() {
	tests := []struct {
		name         string
		expectError  bool
		validateFunc func(string)
	}{
		{
			name:        "returns system hostname",
			expectError: false,
			validateFunc: func(hostname string) {
				s.NotEmpty(hostname, "Local hostname should not be empty")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			hostname, err := job.GetLocalHostname()

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(hostname)
				}
			}
		})
	}
}

func (s *HostnamePublicTestSuite) TestGetAgentHostnameWithProvider() {
	tests := []struct {
		name               string
		configuredHostname string
		setupProvider      func() job.HostnameProvider
		validateFunc       func(string, error)
	}{
		{
			name:               "configured hostname bypasses provider",
			configuredHostname: "configured-agent",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				// Provider is not called when hostname is pre-configured.
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("configured-agent", hostname)
			},
		},
		{
			name:               "empty config uses provider successfully",
			configuredHostname: "",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("system-host", nil)
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("system-host", hostname)
			},
		},
		{
			name:               "empty config with provider error returns unknown",
			configuredHostname: "",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("", errors.New("provider error"))
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("unknown", hostname)
			},
		},
		{
			name:               "empty config with empty hostname returns unknown",
			configuredHostname: "",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("", nil)
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("unknown", hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(job.GetAgentHostnameWithProvider(
				tt.configuredHostname,
				tt.setupProvider(),
			))
		})
	}
}

func (s *HostnamePublicTestSuite) TestGetLocalHostnameWithProvider() {
	tests := []struct {
		name          string
		setupProvider func() job.HostnameProvider
		validateFunc  func(string, error)
	}{
		{
			name: "successful hostname retrieval",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("test-host", nil)
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("test-host", hostname)
			},
		},
		{
			name: "provider error",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("", errors.New("hostname error"))
				return m
			},
			validateFunc: func(_ string, err error) {
				s.Error(err)
			},
		},
		{
			name: "empty hostname from provider",
			setupProvider: func() job.HostnameProvider {
				m := hostnamemocks.NewMockHostnameProvider(s.mockCtrl)
				m.EXPECT().Hostname().Return("", nil)
				return m
			},
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.Equal("", hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(job.GetLocalHostnameWithProvider(tt.setupProvider()))
		})
	}
}

func (s *HostnamePublicTestSuite) TestGopsutilHostnameProvider() {
	tests := []struct {
		name         string
		validateFunc func(string, error)
	}{
		{
			name: "should return non-empty hostname without error",
			validateFunc: func(hostname string, err error) {
				s.NoError(err)
				s.NotEmpty(hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			provider := job.ExportNewGopsutilHostnameProvider()
			hostname, err := provider.Hostname()

			tt.validateFunc(hostname, err)
		})
	}
}

func (s *HostnamePublicTestSuite) TestHostnameProviderInterface() {
	tests := []struct {
		name     string
		provider job.HostnameProvider
	}{
		{
			name:     "gopsutilHostnameProvider implements interface",
			provider: job.ExportNewGopsutilHostnameProvider(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_ = tt.provider
		})
	}
}

func (s *HostnamePublicTestSuite) TestGopsutilHostnameProviderError() {
	tests := []struct {
		name         string
		validateFunc func(string, error)
	}{
		{
			name: "returns error when host.Info fails",
			validateFunc: func(hostname string, err error) {
				s.Error(err)
				s.Empty(hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			job.SetHostInfoFn(func() (*hostpkg.InfoStat, error) {
				return nil, errors.New("host info failed")
			})
			defer job.ResetHostInfoFn()

			provider := job.ExportNewGopsutilHostnameProvider()
			tt.validateFunc(provider.Hostname())
		})
	}
}

func TestHostnamePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnamePublicTestSuite))
}
