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
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execMocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netinfo"
)

type GetRoutesDarwinPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func (suite *GetRoutesDarwinPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *GetRoutesDarwinPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *GetRoutesDarwinPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *GetRoutesDarwinPublicTestSuite) TestGetRoutes() {
	tests := []struct {
		name             string
		routeContent     string
		readerErr        bool
		useDefaultReader bool
		execMockErr      bool
		wantErr          bool
		validateFunc     func(routes []netinfo.RouteResult)
	}{
		{
			name: "when typical macOS route table",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
127                127.0.0.1          UCS         lo0
127.0.0.1          127.0.0.1          UH          lo0
192.168.1/24       link#6             UCS         en0
192.168.1.100      a4:83:e7:1a:2b:3c  UHLWIi      lo0

Internet6:
Destination        Gateway            Flags     Netif Expire
::1                ::1                UHL         lo0
`,
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 5)

				suite.Equal("default", routes[0].Destination)
				suite.Equal("192.168.1.1", routes[0].Gateway)
				suite.Equal("UGScg", routes[0].Flags)
				suite.Equal("en0", routes[0].Interface)

				suite.Equal("127", routes[1].Destination)
				suite.Equal("127.0.0.1", routes[1].Gateway)
				suite.Equal("lo0", routes[1].Interface)

				suite.Equal("192.168.1/24", routes[3].Destination)
				suite.Equal("link#6", routes[3].Gateway)
				suite.Equal("en0", routes[3].Interface)
			},
		},
		{
			name: "when multiple default routes",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
default            10.0.0.1           UGScIg      en1
10.0.0/24          link#7             UCS         en1
`,
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 3)
				suite.Equal("en0", routes[0].Interface)
				suite.Equal("en1", routes[1].Interface)
			},
		},
		{
			name: "when IPv6 lines are skipped",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0

Internet6:
Destination        Gateway            Flags     Netif Expire
default            fe80::1%en0        UGcg        en0
::1                ::1                UHL         lo0
`,
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Equal("default", routes[0].Destination)
				suite.Equal("en0", routes[0].Interface)
			},
		},
		{
			name:         "when no IPv4 routing table found",
			routeContent: "Routing tables\n\nInternet6:\nDestination  Gateway  Flags  Netif Expire\n",
			wantErr:      true,
		},
		{
			name: "when Internet6 appears before header in IPv4 section",
			routeContent: `Routing tables

Internet:
Internet6:
Destination        Gateway            Flags     Netif Expire
`,
			wantErr: true,
		},
		{
			name:         "when empty output",
			routeContent: "",
			wantErr:      true,
		},
		{
			name:      "when reader returns error",
			readerErr: true,
			wantErr:   true,
		},
		{
			name: "when line has too few fields",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
bad
`,
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 1)
				suite.Equal("default", routes[0].Destination)
			},
		},
		{
			name:             "when using default route reader",
			useDefaultReader: true,
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
127                127.0.0.1          UCS         lo0
`,
			validateFunc: func(routes []netinfo.RouteResult) {
				suite.Require().Len(routes, 2)
				suite.Equal("default", routes[0].Destination)
				suite.Equal("192.168.1.1", routes[0].Gateway)
				suite.Equal("en0", routes[0].Interface)
			},
		},
		{
			name:             "when default route reader errors",
			useDefaultReader: true,
			execMockErr:      true,
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := execMocks.NewPlainMockManager(suite.ctrl)

			if tc.useDefaultReader {
				if tc.execMockErr {
					mock.EXPECT().
						RunCmd("netstat", []string{"-rn"}).
						Return("", assert.AnError)
				} else {
					mock.EXPECT().
						RunCmd("netstat", []string{"-rn"}).
						Return(tc.routeContent, nil)
				}
			}

			d := netinfo.NewDarwinProvider(mock)

			if !tc.useDefaultReader {
				if tc.readerErr {
					d.RouteReaderFn = func() (io.ReadCloser, error) {
						return nil, assert.AnError
					}
				} else {
					content := tc.routeContent
					d.RouteReaderFn = func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader(content)), nil
					}
				}
			}

			got, err := d.GetRoutes()

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

func (suite *GetRoutesDarwinPublicTestSuite) TestGetPrimaryInterface() {
	tests := []struct {
		name             string
		routeContent     string
		readerErr        bool
		useDefaultReader bool
		wantErr          bool
		validateFunc     func(iface string)
	}{
		{
			name: "when default route exists",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
127                127.0.0.1          UCS         lo0
`,
			validateFunc: func(iface string) {
				suite.Equal("en0", iface)
			},
		},
		{
			name: "when no default route exists",
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
127                127.0.0.1          UCS         lo0
192.168.1/24       link#6             UCS         en0
`,
			wantErr: true,
		},
		{
			name:      "when reader returns error",
			readerErr: true,
			wantErr:   true,
		},
		{
			name:         "when empty output",
			routeContent: "",
			wantErr:      true,
		},
		{
			name:             "when using default route reader",
			useDefaultReader: true,
			routeContent: `Routing tables

Internet:
Destination        Gateway            Flags     Netif Expire
default            192.168.1.1        UGScg       en0
127                127.0.0.1          UCS         lo0
`,
			validateFunc: func(iface string) {
				suite.Equal("en0", iface)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := execMocks.NewPlainMockManager(suite.ctrl)

			if tc.useDefaultReader {
				mock.EXPECT().
					RunCmd("netstat", []string{"-rn"}).
					Return(tc.routeContent, nil)
			}

			d := netinfo.NewDarwinProvider(mock)

			if !tc.useDefaultReader {
				if tc.readerErr {
					d.RouteReaderFn = func() (io.ReadCloser, error) {
						return nil, assert.AnError
					}
				} else {
					content := tc.routeContent
					d.RouteReaderFn = func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader(content)), nil
					}
				}
			}

			got, err := d.GetPrimaryInterface()

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

func (suite *GetRoutesDarwinPublicTestSuite) TestNewDarwinProvider() {
	tests := []struct {
		name         string
		setupMock    func() func() ([]net.Interface, error)
		wantErr      bool
		validateFunc func(result []netinfo.InterfaceResult)
	}{
		{
			name: "when factory wires GetInterfaces correctly",
			setupMock: func() func() ([]net.Interface, error) {
				return func() ([]net.Interface, error) {
					return []net.Interface{
						{
							Index:        2,
							MTU:          1500,
							Name:         "en0",
							HardwareAddr: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
							Flags:        net.FlagUp | net.FlagBroadcast,
						},
					}, nil
				}
			},
			validateFunc: func(result []netinfo.InterfaceResult) {
				suite.Require().Len(result, 1)
				suite.Equal("en0", result[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := execMocks.NewPlainMockManager(suite.ctrl)
			d := netinfo.NewDarwinProvider(mock)

			d.InterfacesFn = tc.setupMock()

			got, err := d.GetInterfaces()

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

func TestGetRoutesDarwinPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GetRoutesDarwinPublicTestSuite))
}
