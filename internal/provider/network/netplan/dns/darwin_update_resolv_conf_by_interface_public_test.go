// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package dns_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execMocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
)

type DarwinUpdateResolvConfByInterfacePublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	logger *slog.Logger
}

func (suite *DarwinUpdateResolvConfByInterfacePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DarwinUpdateResolvConfByInterfacePublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DarwinUpdateResolvConfByInterfacePublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DarwinUpdateResolvConfByInterfacePublicTestSuite) TestUpdateResolvConfByInterface() {
	tests := []struct {
		name         string
		validateFunc func(*dns.UpdateResult, error)
	}{
		{
			name: "returns ErrUnsupported on Darwin",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			mock := execMocks.NewPlainMockManager(suite.ctrl)

			darwin := dns.NewDarwinProvider(suite.logger, mock)
			tt.validateFunc(darwin.UpdateResolvConfByInterface(
				[]string{"8.8.8.8"},
				[]string{"example.com"},
				"en0",
				false,
			))
		})
	}
}

func (suite *DarwinUpdateResolvConfByInterfacePublicTestSuite) TestDeleteNetplanConfig() {
	tests := []struct {
		name         string
		validateFunc func(bool, error)
	}{
		{
			name: "returns ErrUnsupported on Darwin",
			validateFunc: func(result bool, err error) {
				suite.Error(err)
				suite.False(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			mock := execMocks.NewPlainMockManager(suite.ctrl)

			darwin := dns.NewDarwinProvider(suite.logger, mock)
			tt.validateFunc(darwin.DeleteNetplanConfig("eth0"))
		})
	}
}

func TestDarwinUpdateResolvConfByInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinUpdateResolvConfByInterfacePublicTestSuite))
}
