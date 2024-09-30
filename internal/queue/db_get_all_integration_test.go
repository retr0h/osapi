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

package queue_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/queue"
	qtesting "github.com/retr0h/osapi/internal/queue/testing"
)

type DBGetAllIntegrationTestSuite struct {
	suite.Suite

	qm queue.Manager
}

func (suite *DBGetAllIntegrationTestSuite) SetupTest() {}

func (suite *DBGetAllIntegrationTestSuite) SetupSubTest() {
	// NOTE(retr0h): Need to use SetupSubTest when using table driven tests
	qm, err := qtesting.SetupDatabase()
	suite.Require().NoError(err)
	suite.qm = qm
}

func (suite *DBGetAllIntegrationTestSuite) TearDownTest() {}

func (suite *DBGetAllIntegrationTestSuite) TestGetAll() {
	tests := []struct {
		name       string
		totalItems int
		limit      int
		offset     int
		want       int
		wantErr    bool
	}{
		{
			name:       "when GetAll Ok",
			totalItems: 3,
			limit:      10,
			offset:     0,
			want:       3,
		},
		{
			name:       "when GetAll handles limit Ok",
			totalItems: 3,
			limit:      2,
			offset:     0,
			want:       2,
		},
		{
			name:       "when GetAll handles offset Ok",
			totalItems: 3,
			limit:      2,
			offset:     2,
			want:       1,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			for i := 0; i < tc.totalItems; i++ {
				msg := fmt.Sprintf("test message from %s iteration %d", suite.T().Name(), i)
				err := suite.qm.Put(context.Background(), []byte(msg))
				assert.NoError(suite.T(), err)
			}

			got, err := suite.qm.GetAll(context.Background(), tc.limit, tc.offset)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), tc.want, len(got))
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDBGetAllIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DBGetAllIntegrationTestSuite))
}
