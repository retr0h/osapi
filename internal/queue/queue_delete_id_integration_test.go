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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/queue"
	"github.com/retr0h/osapi/internal/queue/helpers"
)

type DeleteByIDIntegrationTestSuite struct {
	suite.Suite

	qm queue.Manager
}

func (suite *DeleteByIDIntegrationTestSuite) SetupTest() {
	qm, err := helpers.SetupDatabase()
	suite.Require().NoError(err)
	suite.qm = qm
}

func (suite *DeleteByIDIntegrationTestSuite) SetupSubTest() {}

func (suite *DeleteByIDIntegrationTestSuite) TearDownTest() {}

func (suite *DeleteByIDIntegrationTestSuite) TestDeleteByIDOk() {
	err := suite.qm.Put(context.Background(), []byte("testMessage"))
	assert.NoError(suite.T(), err)

	items, err := suite.qm.GetAll(context.Background(), 10, 0)
	assert.NoError(suite.T(), err)

	item := items[0]
	err = suite.qm.DeleteByID(context.Background(), item.ID)
	assert.NoError(suite.T(), err)
}

func (suite *DeleteByIDIntegrationTestSuite) TestDeleteByIDErrorsWhenQueueDeleteFails() {
	suite.T().Skip("https://github.com/maragudk/goqite/issues/58")

	err := suite.qm.Put(context.Background(), []byte("testMessage"))
	assert.NoError(suite.T(), err)

	messageID := "non-existent"
	err = suite.qm.DeleteByID(context.Background(), messageID)
	assert.Error(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDeleteByIDIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteByIDIntegrationTestSuite))
}
