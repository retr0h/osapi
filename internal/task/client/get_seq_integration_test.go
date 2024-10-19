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

	"github.com/stretchr/testify/suite"

	customerrors "github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/task/client"
	clienttest "github.com/retr0h/osapi/internal/task/client/testing"
	"github.com/retr0h/osapi/internal/task/server"
)

type GetMessageBySeqIntegrationTestSuite struct {
	suite.Suite

	server *server.Server
	client *client.Client
}

func (suite *GetMessageBySeqIntegrationTestSuite) SetupTest() {
	suite.server = clienttest.NewServer()
	err := suite.server.Start()
	suite.Require().NoError(err)

	c, err := clienttest.NewClient()
	suite.Require().NoError(err)
	suite.client = c
}

func (suite *GetMessageBySeqIntegrationTestSuite) SetupSubTest() {}

func (suite *GetMessageBySeqIntegrationTestSuite) TearDownTest() {
	defer suite.server.Stop()
}

func (suite *GetMessageBySeqIntegrationTestSuite) TestGetMessageBySeq() {
	tests := []struct {
		name        string
		data        []byte
		seq         uint64
		want        string
		wantErr     bool
		wantErrType error
	}{
		{
			name:    "when GetMessageBySeq Ok",
			data:    []byte("test message"),
			want:    "test message",
			wantErr: false,
		},
		{
			name:        "when GetMessageBySeq fails due to non-existent message",
			seq:         uint64(666),
			want:        "test message",
			wantErr:     true,
			wantErrType: customerrors.NewNotFoundError("no item found with ID 666"),
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			var seq uint64
			var err error

			if len(tc.data) > 0 {
				seq, err = suite.client.PublishToStream(ctx, tc.data)
				suite.Require().NoError(err)
			} else {
				seq = tc.seq
			}

			got, err := suite.client.GetMessageBySeq(ctx, seq)

			if !tc.wantErr {
				suite.NoError(err)
				suite.Equal(tc.want, string(got.Data))
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestGetMessageBySeqIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(GetMessageBySeqIntegrationTestSuite))
}
