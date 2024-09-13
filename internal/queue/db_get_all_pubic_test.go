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
	"database/sql"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/queue"
	"github.com/retr0h/osapi/internal/queue/mocks"
)

type DBGetAllTestSuite struct {
	suite.Suite
	db   *sql.DB
	mock sqlmock.Sqlmock

	fixedCreated time.Time
	updated      time.Time
	timeout      time.Time
}

func (suite *DBGetAllTestSuite) SetupTest() {
	db, mock, err := sqlmock.New()
	suite.Require().NoError(err)
	suite.db = db
	suite.mock = mock

	suite.fixedCreated, _ = mocks.GetFixedTime()
	suite.timeout = suite.fixedCreated.Add(time.Hour)
	suite.updated = suite.fixedCreated.Add(time.Minute)
}

func (suite *DBGetAllTestSuite) TearDownTest() {
	suite.Require().NoError(suite.mock.ExpectationsWereMet())
}

func (suite *DBGetAllTestSuite) TestGetAll() {
	tests := []struct {
		name        string
		setupMock   func()
		want        []queue.Item
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetAll Ok",
			setupMock: func() {
				query := regexp.QuoteMeta(`
					SELECT id, created, updated, queue, body, timeout, received
					FROM goqite
					LIMIT ? OFFSET ?`)

				rows := sqlmock.NewRows([]string{"id", "created", "updated", "queue", "body", "timeout", "received"}).
					AddRow("1", suite.fixedCreated, suite.updated, "test-queue", "test-body", suite.timeout, 1)

				suite.mock.ExpectQuery(query).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			want: []queue.Item{
				{
					ID:       "1",
					Created:  suite.fixedCreated,
					Updated:  suite.updated,
					Queue:    "test-queue",
					Body:     "test-body",
					Timeout:  suite.timeout,
					Received: 1,
				},
			},
			wantErr: false,
		},

		{
			name: "when GetAll fails to execute statement",
			setupMock: func() {
				query := regexp.QuoteMeta(`
					SELECT id, created, updated, queue, body, timeout, received
					FROM goqite
					LIMIT ? OFFSET ?`)

				suite.mock.ExpectQuery(query).
					WithArgs(10, 0).
					WillReturnError(fmt.Errorf("query execution failed"))
			},
			want:        nil,
			wantErr:     true,
			wantErrType: fmt.Errorf("query execution failed"),
		},
		{
			name: "when GetAll fails to execute scan",
			setupMock: func() {
				query := regexp.QuoteMeta(`
					SELECT id, created, updated, queue, body, timeout, received
					FROM goqite
					LIMIT ? OFFSET ?`)

				rows := sqlmock.NewRows([]string{"id", "created", "updated", "queue", "body", "timeout", "received"}).
					AddRow("1", suite.fixedCreated, suite.updated, "test-queue", "test-body", suite.timeout, 1)

				rows.RowError(0, fmt.Errorf("scan error"))

				suite.mock.ExpectQuery(query).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			want:        nil,
			wantErr:     true,
			wantErrType: fmt.Errorf("scan error"),
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			q := &queue.Queue{DB: suite.db}
			got, err := q.GetAll(context.Background(), 10, 0)

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
func TestDBGetAllTestSuite(t *testing.T) {
	suite.Run(t, new(DBGetAllTestSuite))
}
