//go:build test
// +build test

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

package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/maragudk/goqite"
)

const (
	// fixedTime pinned time used by test suite.
	fixedTime     = "2024-09-10T12:00:00Z"
	itemBody      = "test body"
	itemMessageID = "message-id"
	itemReceived  = 5
)

// MockQueue is a mock implementation of the Queue interface for testing.
type MockQueue struct {
	GetAllFunc  func() ([]Item, error)
	GetByIDFunc func() (*Item, error)
}

// NewDefaultMockQueue creates a MockQueue with default return values.
func NewDefaultMockQueue() *MockQueue {
	fixedCreated, _ := GetMockFixedTime()
	timeout := fixedCreated.Add(time.Hour)
	updated := fixedCreated.Add(time.Minute)

	return &MockQueue{
		GetAllFunc: func() ([]Item, error) {
			return []Item{
				{
					Body:     itemBody,
					ID:       itemMessageID,
					Received: itemReceived,
					Created:  fixedCreated,
					Timeout:  timeout,
					Updated:  updated,
				},
			}, nil
		},
		GetByIDFunc: func() (*Item, error) {
			return &Item{
				Body:     itemBody,
				ID:       itemMessageID,
				Received: itemReceived,
				Created:  fixedCreated,
				Timeout:  timeout,
				Updated:  updated,
			}, nil
		},
	}
}

// SetupQueue mocked for tests.
func (mq *MockQueue) SetupQueue() error {
	// q.Queue = goqite.New(goqite.NewOpts{
	// 	DB:   q.DB,
	// 	Name: "osapi_jobs",
	// })
	return nil
}

// Get mocked for tests.
func (mq *MockQueue) Get(
	_ context.Context,
) (*goqite.Message, error) {
	return nil, nil
}

// GetAll mocked for tests
func (mq *MockQueue) GetAll(
	_ context.Context,
	_ int,
	_ int,
) ([]Item, error) {
	return mq.GetAllFunc()
}

// GetByID mocked for tests
func (mq *MockQueue) GetByID(
	_ context.Context,
	_ string,
) (*Item, error) {
	return mq.GetByIDFunc()
}

// Put mocked for tests.
func (mq *MockQueue) Put(
	_ context.Context,
	_ []byte,
) error {
	return nil
}

// Delete mocked for tests.
func (mq *MockQueue) Delete(
	_ context.Context,
	_ goqite.ID,
) error {
	return nil
}

// GetMockFixedTime parses and returns the fixedTime constant as a time.Time value.
// If the parsing fails, it returns an error.
func GetMockFixedTime() (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, fixedTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid fixed time format: %w", err)
	}
	return parsedTime, nil
}

// GetMockItemBody returns the value of itemBody.
func GetMockItemBody() string {
	return itemBody
}

// GetMockItemMessageID returns the value of itemMessageID.
func GetMockItemMessageID() string {
	return itemMessageID
}

// GetMockItemReceived returns the value of itemReceived.
func GetMockItemReceived() int {
	return itemReceived
}
