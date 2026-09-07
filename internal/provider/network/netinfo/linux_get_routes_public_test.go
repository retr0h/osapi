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

package netinfo_test

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/network/netinfo"
)

type GetRoutesPublicTestSuite struct {
	suite.Suite
}

func (suite *GetRoutesPublicTestSuite) SetupTest() {}

func (suite *GetRoutesPublicTestSuite) TearDownTest() {}

func (suite *GetRoutesPublicTestSuite) TestGetRoutes() {
	tests := []struct {
		name             string
		routeContent     string
		readerErr        bool
		useDefaultReader bool
		wantErr          bool
		skipErrCheck     bool
		validateFunc     func(routes []netinfo.RouteResult)
	}{
		{
			name: "when typical route table with default and subnet routes",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
				"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 2)

				suite.Equal("0.0.0.0", routes[0].Destination)
				suite.Equal("192.168.1.1", routes[0].Gateway)
				suite.Equal("eth0", routes[0].Interface)
				suite.Equal("/0", routes[0].Mask)
				suite.Equal(100, routes[0].Metric)
				suite.Equal("0003", routes[0].Flags)

				suite.Equal("192.168.1.0", routes[1].Destination)
				suite.Equal("0.0.0.0", routes[1].Gateway)
				suite.Equal("eth0", routes[1].Interface)
				suite.Equal("/24", routes[1].Mask)
			},
		},
		{
			name:         "when route table is empty (header only)",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Empty(routes)
			},
		},
		{
			name:      "when reader returns error",
			readerErr: true,
			wantErr:   true,
		},
		{
			name:         "when route table has no header",
			routeContent: "",
			wantErr:      true,
		},
		{
			name: "when line has too few fields",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t00000000\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Empty(routes)
			},
		},
		{
			name: "when hex IP contains invalid characters",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\tZZZZZZZZ\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Empty(routes[0].Destination)
			},
		},
		{
			name: "when hex IP has wrong length",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t0000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Empty(routes[0].Destination)
			},
		},
		{
			name: "when hex mask contains invalid characters",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\tXXXXXXXX\t0\t0\t0\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Empty(routes[0].Mask)
			},
		},
		{
			name: "when hex mask has wrong length",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00FF\t0\t0\t0\n",
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Empty(routes[0].Mask)
			},
		},
		{
			name:             "when using default route reader",
			useDefaultReader: true,
			skipErrCheck:     true, // succeeds on Linux, errors on macOS
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			l := netinfo.NewLinuxProvider()

			if !tc.useDefaultReader {
				if tc.readerErr {
					l.RouteReaderFn = func() (io.ReadCloser, error) {
						return nil, assert.AnError
					}
				} else {
					content := tc.routeContent
					l.RouteReaderFn = func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader(content)), nil
					}
				}
			}

			got, err := l.GetRoutes()

			if tc.skipErrCheck {
				// Succeeds on Linux, errors on macOS — both are valid
				return
			}
			if tc.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				if tc.validateFunc != nil {
					tc.validateFunc(got)
				}
			}
		})
	}
}

func (suite *GetRoutesPublicTestSuite) TestGetPrimaryInterface() {
	tests := []struct {
		name         string
		routeContent string
		readerErr    bool
		wantErr      bool
		validateFunc func(iface string)
	}{
		{
			name: "when default route exists",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
				"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n",
			validateFunc: func(iface string) {
				suite.Equal("eth0", iface)
			},
		},
		{
			name: "when no default route exists",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n",
			wantErr: true,
		},
		{
			name:      "when reader returns error",
			readerErr: true,
			wantErr:   true,
		},
		{
			name:         "when route table has no header",
			routeContent: "",
			wantErr:      true,
		},
		{
			name: "when line has too few fields",
			routeContent: "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
				"eth0\n",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			l := netinfo.NewLinuxProvider()

			if tc.readerErr {
				l.RouteReaderFn = func() (io.ReadCloser, error) {
					return nil, assert.AnError
				}
			} else {
				content := tc.routeContent
				l.RouteReaderFn = func() (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(content)), nil
				}
			}

			got, err := l.GetPrimaryInterface()

			if tc.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				if tc.validateFunc != nil {
					tc.validateFunc(got)
				}
			}
		})
	}
}

func TestGetRoutesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GetRoutesPublicTestSuite))
}
