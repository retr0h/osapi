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

package cli_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/cli"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func TestValidatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

// exitSentinel is used to distinguish a controlled panic from osExit
// from an unexpected panic during test execution.
type exitSentinel struct{}

type ValidateDistributionPublicTestSuite struct {
	suite.Suite
}

func (suite *ValidateDistributionPublicTestSuite) TearDownTest() {
	cli.ResetOsExit()
	cli.ResetHostInfoFn()
	_ = os.Unsetenv("IGNORE_LINUX")
}

func (suite *ValidateDistributionPublicTestSuite) TestValidateDistribution() {
	tests := []struct {
		name         string
		ignoreLinux  bool
		hostInfoFn   func() (*host.InfoStat, error)
		validateFunc func(bool)
	}{
		{
			name: "when host info fails calls LogFatal",
			hostInfoFn: func() (*host.InfoStat, error) {
				return nil, fmt.Errorf("host info failed")
			},
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), true, got)
			},
		},
		{
			name:        "when IGNORE_LINUX is set returns early",
			ignoreLinux: true,
			hostInfoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "darwin",
					PlatformVersion: "14.0",
				}, nil
			},
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), false, got)
			},
		},
		{
			name: "when supported version does not exit",
			hostInfoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "ubuntu",
					PlatformVersion: "24.04",
				}, nil
			},
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), false, got)
			},
		},
		{
			name: "when unsupported version calls LogFatal",
			hostInfoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "centos",
					PlatformVersion: "8",
				}, nil
			},
			validateFunc: func(got bool) {
				assert.Equal(suite.T(), true, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.ignoreLinux {
				_ = os.Setenv("IGNORE_LINUX", "1")
			} else {
				_ = os.Unsetenv("IGNORE_LINUX")
			}

			var exited bool
			cli.SetOsExit(func(_ int) {
				exited = true
				panic(exitSentinel{})
			})
			cli.SetHostInfoFn(tc.hostInfoFn)

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			func() {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(exitSentinel); !ok {
							panic(r)
						}
					}
				}()
				cli.ValidateDistribution(logger)
			}()

			tc.validateFunc(exited)
		})
	}
}

func TestValidateDistributionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidateDistributionPublicTestSuite))
}

func (suite *ValidatePublicTestSuite) TestIsOSFamilySupported() {
	tests := []struct {
		name         string
		distro       string
		version      string
		wantFamily   string
		validateFunc func(bool)
	}{
		{
			name:       "when darwin is supported with any version",
			distro:     "darwin",
			version:    "14.0",
			wantFamily: "Darwin",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when debian 12 is supported",
			distro:     "debian",
			version:    "12",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when debian 12 point release is supported",
			distro:     "debian",
			version:    "12.13",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when debian 13 is supported",
			distro:     "debian",
			version:    "13",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when ubuntu 20.04 is supported",
			distro:     "ubuntu",
			version:    "20.04",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when ubuntu 22.04 is supported",
			distro:     "ubuntu",
			version:    "22.04",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when ubuntu 24.04 is supported",
			distro:     "ubuntu",
			version:    "24.04",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when Ubuntu with uppercase is supported",
			distro:     "Ubuntu",
			version:    "24.04",
			wantFamily: "Debian",
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name:       "when unsupported distro returns false",
			distro:     "centos",
			version:    "8",
			wantFamily: "",
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name:       "when unsupported version returns false",
			distro:     "ubuntu",
			version:    "18.04",
			wantFamily: "",
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name:       "when empty distro returns false",
			distro:     "",
			version:    "24.04",
			wantFamily: "",
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name:       "when empty version returns false",
			distro:     "ubuntu",
			version:    "",
			wantFamily: "",
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			family, ok := cli.IsOSFamilySupported(tc.distro, tc.version)

			suite.Equal(tc.wantFamily, family)
			tc.validateFunc(ok)
		})
	}
}
