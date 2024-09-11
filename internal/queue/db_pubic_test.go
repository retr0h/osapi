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
	"time"

	"github.com/retr0h/osapi/internal/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DBPublicTestSuite struct {
	suite.Suite

	fixedCreated time.Time
	updated      time.Time
	timeout      time.Time
	body         string
	messageID    string
	received     int
}

func (suite *DBPublicTestSuite) SetupTest() {
	suite.fixedCreated, _ = queue.GetMockFixedTime()
	suite.timeout = suite.fixedCreated.Add(time.Hour)
	suite.updated = suite.fixedCreated.Add(time.Minute)
	suite.body = queue.GetMockItemBody()
	suite.messageID = queue.GetMockItemMessageID()
	suite.received = queue.GetMockItemReceived()
}

func (suite *DBPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DBPublicTestSuite) TearDownTest() {}

func (suite *DBPublicTestSuite) TestDB() {
	tests := []struct {
		name      string
		setupMock func() *queue.MockQueue
		fn        func(*queue.MockQueue) (interface{}, error)
		want      interface{}
		wantErr   bool
	}{
		{
			name: "when GetAll Ok",
			setupMock: func() *queue.MockQueue {
				mock := queue.NewDefaultMockQueue()
				return mock
			},
			fn: func(m *queue.MockQueue) (interface{}, error) {
				return m.GetAll(context.Background(), 0, 1)
			},
			want: []queue.Item{
				{
					Body:     suite.body,
					ID:       suite.messageID,
					Received: suite.received,
					Created:  suite.fixedCreated,
					Timeout:  suite.timeout,
					Updated:  suite.updated,
				},
			},
			wantErr: false,
		},
		{
			name: "when GetAll errors",
			setupMock: func() *queue.MockQueue {
				mock := queue.NewDefaultMockQueue()
				mock.GetAllFunc = func() ([]queue.Item, error) {
					return nil, fmt.Errorf("GetAll error")
				}
				return mock
			},
			fn: func(m *queue.MockQueue) (interface{}, error) {
				return m.GetAll(context.Background(), 0, 1)
			},
			wantErr: true,
		},
		{
			name: "when GetByID Ok",
			setupMock: func() *queue.MockQueue {
				mock := queue.NewDefaultMockQueue()
				return mock
			},
			fn: func(m *queue.MockQueue) (interface{}, error) {
				return m.GetByID(context.Background(), suite.messageID)
			},
			want: &queue.Item{
				Body:     suite.body,
				ID:       suite.messageID,
				Received: suite.received,
				Created:  suite.fixedCreated,
				Timeout:  suite.timeout,
				Updated:  suite.updated,
			},
			wantErr: false,
		},
		{
			name: "when GetByID errors",
			setupMock: func() *queue.MockQueue {
				mock := queue.NewDefaultMockQueue()
				mock.GetByIDFunc = func() (*queue.Item, error) {
					return nil, fmt.Errorf("GetByID error")
				}
				return mock
			},
			fn: func(m *queue.MockQueue) (interface{}, error) {
				return m.GetByID(context.Background(), suite.messageID)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			got, err := tc.fn(mock)

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDBPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DBPublicTestSuite))
}
