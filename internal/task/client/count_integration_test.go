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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task/client"
	clienttest "github.com/retr0h/osapi/internal/task/client/testing"
	"github.com/retr0h/osapi/internal/task/server"
)

type CountStreamMessagesIntegrationTestSuite struct {
	suite.Suite

	server *server.Server
	client *client.Client
}

func (suite *CountStreamMessagesIntegrationTestSuite) SetupTest() {
	suite.server = clienttest.NewServer()
	err := suite.server.Start()
	suite.Require().NoError(err)

	c, err := clienttest.NewClient()
	suite.Require().NoError(err)
	suite.client = c
}

func (suite *CountStreamMessagesIntegrationTestSuite) SetupSubTest() {}

func (suite *CountStreamMessagesIntegrationTestSuite) TearDownTest() {
	defer suite.server.Stop()
}

func (suite *CountStreamMessagesIntegrationTestSuite) TestCountStreamMessages() {
	tests := []struct {
		name          string
		totalMessages int
		want          int
		wantErr       bool
		wantErrType   error
	}{
		{
			name:          "when CountStreamMessages Ok",
			totalMessages: 3,
			want:          3,
			wantErr:       false,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			for i := 0; i < tc.totalMessages; i++ {
				data := []byte("test message")
				_, err := suite.client.PublishToStream(ctx, data)
				suite.Require().NoError(err)
			}

			got, err := suite.client.CountStreamMessages(ctx)

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestCountStreamMessagesIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CountStreamMessagesIntegrationTestSuite))
}
