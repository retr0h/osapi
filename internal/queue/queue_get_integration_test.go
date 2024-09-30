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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/queue"
	"github.com/retr0h/osapi/internal/queue/mocks"
	qtesting "github.com/retr0h/osapi/internal/queue/testing"
)

type QueueGetIntegrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	qm queue.Manager
}

func (suite *QueueGetIntegrationTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	qm, err := qtesting.SetupDatabase()
	suite.Require().NoError(err)
	suite.qm = qm
}

func (suite *QueueGetIntegrationTestSuite) SetupSubTest() {}

func (suite *QueueGetIntegrationTestSuite) TearDownTest() {}

func (suite *QueueGetIntegrationTestSuite) TestGetOk() {
	for i := 0; i < 3; i++ {
		msg := fmt.Sprintf("test message from %s iteration %d", suite.T().Name(), i)
		err := suite.qm.Put(context.Background(), []byte(msg))
		assert.NoError(suite.T(), err)
	}

	got, err := suite.qm.Get(context.Background())
	assert.NoError(suite.T(), err)

	want := fmt.Sprintf("test message from %s iteration %d", suite.T().Name(), 0)
	assert.Equal(suite.T(), want, string(got.Body))
}

func (suite *QueueGetIntegrationTestSuite) TestPutErrorsWhenQueueReceiveFails() {
	mock := mocks.NewPlainMockMessageProcessor(suite.ctrl)
	mock.EXPECT().Receive(context.Background()).
		Return(nil, assert.AnError)

	suite.qm.SetQueue(mock)

	_, err := suite.qm.Get(context.Background())
	assert.Error(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestQueueGetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QueueGetIntegrationTestSuite))
}
