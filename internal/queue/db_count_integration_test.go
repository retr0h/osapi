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

type DBCountIntegrationTestSuite struct {
	suite.Suite

	qm queue.Manager
}

func (suite *DBCountIntegrationTestSuite) SetupTest() {
	qm, err := qtesting.SetupDatabase()
	suite.Require().NoError(err)
	suite.qm = qm
}

func (suite *DBCountIntegrationTestSuite) SetupSubTest() {}

func (suite *DBCountIntegrationTestSuite) TearDownTest() {}

func (suite *DBCountIntegrationTestSuite) TestCountOk() {
	for i := 0; i < 3; i++ {
		msg := fmt.Sprintf("test message from %s iteration %d", suite.T().Name(), i)
		err := suite.qm.Put(context.Background(), []byte(msg))
		assert.NoError(suite.T(), err)
	}

	got, err := suite.qm.Count(context.Background())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, got)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDBCountIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DBCountIntegrationTestSuite))
}
