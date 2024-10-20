// Copyright (c) 2024 John Dewey

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

package client_test

import (
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task/client"
	clienttest "github.com/retr0h/osapi/internal/task/client/testing"
)

type ConnectPubicTestSuite struct {
	suite.Suite

	client *client.Client
}

func (suite *ConnectPubicTestSuite) SetupTest() {
	suite.client = clienttest.NewClientWithoutConnect()
}

func (suite *ConnectPubicTestSuite) SetupSubTest() {}

func (suite *ConnectPubicTestSuite) TearDownTest() {}

func (suite *ConnectPubicTestSuite) TestConnect() {
	tests := []struct {
		name               string
		setupConnectMock   func() func(string, ...nats.Option) (*nats.Conn, error)
		setupJetStreamMock func() func(*nats.Conn) (jetstream.JetStream, error)
		wantErr            bool
		wantErrType        error
	}{
		{
			name: "when Connect Ok",
			setupConnectMock: func() func(string, ...nats.Option) (*nats.Conn, error) {
				return func(string, ...nats.Option) (*nats.Conn, error) {
					return &nats.Conn{}, nil
				}
			},
			setupJetStreamMock: nil,
			wantErr:            false,
		},
		{
			name: "when nats.Connect errors",
			setupConnectMock: func() func(string, ...nats.Option) (*nats.Conn, error) {
				return func(string, ...nats.Option) (*nats.Conn, error) {
					return nil, assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when jetstream.New errors",
			setupConnectMock: func() func(string, ...nats.Option) (*nats.Conn, error) {
				return func(string, ...nats.Option) (*nats.Conn, error) {
					return &nats.Conn{}, nil
				}
			},
			setupJetStreamMock: func() func(*nats.Conn) (jetstream.JetStream, error) {
				return func(*nats.Conn) (jetstream.JetStream, error) {
					return nil, assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.client.ConnectFn = tc.setupConnectMock()
			if tc.setupJetStreamMock != nil {
				suite.client.JetStreamFn = tc.setupJetStreamMock()
			}

			err := suite.client.Connect()

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.wantErrType.Error())
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestConnectPubicTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectPubicTestSuite))
}
